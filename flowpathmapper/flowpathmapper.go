// Copyright 2015 Dominique Feyer <dfeyer@ttree.ch>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flowpathmapper

import (
	"github.com/dfeyer/flow-debugproxy/config"
	"github.com/dfeyer/flow-debugproxy/errorhandler"
	"github.com/dfeyer/flow-debugproxy/pathmapperfactory"
	"github.com/dfeyer/flow-debugproxy/pathmapping"

	log "github.com/Sirupsen/logrus"

	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	h                = "%s"
	framework        = "flow"
	cachePathPattern = "@base@/Data/Temporary/@context@/Cache/Code/Flow_Object_Classes/@filename@.php"
)

var (
	regexpPhpFile         = regexp.MustCompile(`(?://)?(/[^ ]*\.php)`)
	regexpFilename        = regexp.MustCompile(`filename=["]?file://(\S+)/Data/Temporary/[^/]*/Cache/Code/Flow_Object_Classes/([^"]*)\.php`)
	regexpPathAndFilename = regexp.MustCompile(`(?m)^# PathAndFilename: (.*)$`)
	regexpPackageClass    = regexp.MustCompile(`(.*?)/Packages/[^/]*/(.*?)/Classes/(.*).php`)
	regexpDot             = regexp.MustCompile(`[\./]`)
)

func init() {
	p := &PathMapper{}
	pathmapperfactory.Register(framework, p)
}

// PathMapper handle the mapping between real code and proxy
type PathMapper struct {
	config      *config.Config
	pathMapping *pathmapping.PathMapping
}

// Initialize the path mapper dependencies
func (p *PathMapper) Initialize(c *config.Config, m *pathmapping.PathMapping) {
	p.config = c
	p.pathMapping = m
}

// ApplyMappingToTextProtocol change file path in xDebug text protocol
func (p *PathMapper) ApplyMappingToTextProtocol(message []byte) []byte {
	return p.doTextPathMapping(message)
}

// ApplyMappingToXML change file path in xDebug XML protocol
func (p *PathMapper) ApplyMappingToXML(message []byte) []byte {
	message = p.doXMLPathMapping(message)

	// update xml length count
	s := strings.Split(string(message), "\x00")
	i, err := strconv.Atoi(s[0])
	errorhandler.PanicHandler(err)
	l := len(s[1])
	if i != l {
		message = bytes.Replace(message, []byte(strconv.Itoa(i)), []byte(strconv.Itoa(l)), 1)
	}

	return message
}

func (p *PathMapper) doTextPathMapping(message []byte) []byte {
	var processedMapping = map[string]string{}
	for _, match := range regexpPhpFile.FindAllStringSubmatch(string(message), -1) {
		originalPath := match[1]
		path := p.mapPath(originalPath)
		log.WithFields(log.Fields{
			"path":          path,
			"original_path": originalPath,
		}).Warn("Process Text Protocol")
		processedMapping[path] = originalPath
	}

	for path, originalPath := range processedMapping {
		message = bytes.Replace(message, []byte(p.getRealFilename(originalPath)), []byte(p.getRealFilename(path)), -1)
	}

	return message
}

func (p *PathMapper) getCachePath(base, filename string) string {
	cachePath := strings.Replace(cachePathPattern, "@base@", base, 1)
	cachePath = strings.Replace(cachePath, "@context@", p.config.Context, 1)
	return strings.Replace(cachePath, "@filename@", filename, 1)
}

func (p *PathMapper) doXMLPathMapping(b []byte) []byte {
	var processedMapping = map[string]string{}
	for _, match := range regexpFilename.FindAllStringSubmatch(string(b), -1) {
		path := p.getCachePath(match[1], match[2])
		if _, ok := processedMapping[path]; ok == false {
			if originalPath, exist := p.pathMapping.Get(path); exist {
				processedMapping[path] = originalPath
				log.WithFields(log.Fields{
					"path":          path,
					"original_path": originalPath,
				}).Debug("Process XML Protocol")
			} else {
				originalPath = p.readOriginalPathFromCache(path)
				processedMapping[path] = originalPath
				log.WithFields(log.Fields{
					"path":          path,
					"original_path": originalPath,
				}).Debug("Process XML Protocol, missing mapping")
			}
		}
	}

	for path, originalPath := range processedMapping {
		path = p.getRealFilename(path)
		originalPath = p.getRealFilename(originalPath)
		b = bytes.Replace(b, []byte(path), []byte(originalPath), -1)
	}

	return b
}

// getRealFilename removes file:// protocol from the given path
func (p *PathMapper) getRealFilename(path string) string {
	return strings.TrimPrefix(path, "file://")
}

func (p *PathMapper) mapPath(originalPath string) string {
	if strings.Contains(originalPath, "/Packages/") {
		log.WithFields(log.Fields{
			"original_path": originalPath,
		}).Debug("Flow package detected")
		cachePath := p.getCachePath(p.buildClassNameFromPath(originalPath))
		realPath := p.getRealFilename(cachePath)
		if _, err := os.Stat(realPath); err == nil {
			return p.setPathMapping(realPath, originalPath)
		}
	}

	return originalPath
}

func (p *PathMapper) setPathMapping(path string, originalPath string) string {
	if p.config.Verbose {
		log.WithFields(log.Fields{
			"path": path,
		}).Info("Our Umpa Lumpa take care of your mapping and they did a great job, they found a proxy for you")
	}

	if p.pathMapping.Has(path) == false {
		p.pathMapping.Set(path, originalPath)
	}
	return path
}

func (p *PathMapper) readOriginalPathFromCache(path string) string {
	dat, err := ioutil.ReadFile(path)
	errorhandler.PanicHandler(err)
	match := regexpPathAndFilename.FindStringSubmatch(string(dat))
	if len(match) == 2 {
		originalPath := match[1]
		if p.config.VeryVerbose {
			log.WithFields(log.Fields{
				"path":          path,
				"original_path": originalPath,
			}).Info("Umpa Lumpa need to work harder, reverse mapping found")
		}
		p.setPathMapping(path, originalPath)
		return originalPath
	}
	return path
}

func (p *PathMapper) buildClassNameFromPath(path string) (string, string) {
	basePath, className := pathToClassPath(path)
	if className == "" {
		// Other (vendor) packages, todo add support for vendor package with Flow proxy class
		log.WithFields(log.Fields{
			"path": path,
		}).Warn("Vendor package detected, class mapping disabled")
	}
	return basePath, className
}

// Convert absolute path to class path (internal use only)
func pathToClassPath(path string) (string, string) {
	var (
		basePath  string
		classPath string
	)
	match := regexpPackageClass.FindStringSubmatch(path)
	if len(match) == 4 {
		// Flow standard packages
		packagePath := regexpDot.ReplaceAllString(match[2], "/")
		classPath = match[3]
		if strings.Contains(classPath, packagePath) == false {
			// Quick'n dirty PSR4 support
			classPath = packagePath + "/" + classPath
		}
		basePath = match[1]
		classPath = regexpDot.ReplaceAllString(classPath, "_")
	} else {
		// Other (vendor) packages, todo add support for vendor package with Flow proxy class
		basePath = path
		classPath = ""
	}
	return basePath, classPath
}
