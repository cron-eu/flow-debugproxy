Flow Framework Debug Proxy for xDebug
-------------------------------------

Flow Framework is a web application platform enabling developers creating
excellent web solutions and bring back the joy of coding. It gives you fast
results. It is a reliable foundation for complex applications.

The biggest pain with Flow Framework come from the the proxy class, the
framework do not execute your own code, but a precompiled version. This is
required for advanced feature, like AOP and the security framework. So working
with Flow is a real pleasure, but adding xDebug in the setup can be a pain.

This project is an xDebug proxy, written in Go, to take care of the mapping
between your PHP file and the proxy class.

Build your own
--------------

    # Get the dependecies
    go get
    # Build
    go build

Run the proxy
-------------

    # Don't forget to change the configuration of your IDE to use port 9010
    flow-debugproxy -vv --framework flow

How to debug the proxy class directly
-------------------------------------

You can disable to path mapping, in this case the proxy do not process xDebug
protocol:

    ./flow-debugproxy --framework dummy

Show help
---------

    ./flow-debugproxy help

Use with Docker
---------------

Use the [official docker image](https://hub.docker.com/r/dfeyer/flow-debugproxy/) and follow the instruction for the configuration.

##### PHP configuration

```
[Xdebug]
zend_extension=/.../xdebug.so
xdebug.remote_enable=1
xdebug.idekey=PHPSTORM
; The IP or name of the proxy container
xdebug.remote_host=debugproxy
; The proxy port (9010 by default, to not have issue is you use PHP FPM, already on port 9000)
xdebug.remote_port=9010
;xdebug.remote_log=/tmp/xdebug.log
```

You can use the `xdebug.remote_log` to debug the protocol between your container and the proxy, it's useful to catch network issues.

##### Docker Compose

This is an incomplete Docker Compose configuration:

```
services:
  debugproxy:
    image: dfeyer/flow-debugproxy:latest
    volumes:
      - .:/data
    environment:
      # This MUST be the IP address of the IDE (your computer)
      - "IDE_IP=192.168.1.130"
      # This is the default value, need to match the xdebug.remote_port on your php.ini
      - "XDEBUG_PORT=9010"
      # Use this to enable verbose debugging on the proxy
      # - "ADDITIONAL_ARGS=-vv --debug"
    networks:
      - backend

  # This is your application containers, you need to link it to the proxy
  app:
    # The proxy need an access to the project files, to be able to do the path mapping
    volumes:
      - .:/data
    links:
      - debugproxy
```

**Options summary:**
* `IDE_IP` The primary local W-/LAN IP of your machine where your IDE runs on
* `IDE_PORT` The Port your IDE is listening for incoming xdebug connections. (The port the debug proxy will try to connect to)
* `XDEBUG_PORT` The port on which xdebug will try to establish a connection (to this container)
* `FRAMEWORK` Currently supported values: `flow` and `dummy`
* `ADDITIONAL_ARGS` For any additional argument like verbosity flags (`-vv`) or debug mode (`--debug`) (or both)

**Debugging the debugger**

Start the debug proxy with verbose flags if it does not connect to your IDE.
The debug proxy does not quit after stopping the process that started it.
You have to kill it in the container manually.

Hint:

If you use the env variable `FLOW_PATH_TEMPORARY_BASE`, please be sure to keep
`Data/Temporary` inside the path, without this the mapper will not detect the
proxy classes.

```
FLOW_PATH_TEMPORARY_BASE=/tmp/flow/Data/Temporary
```

Using with --framework dummy
----------------------------

If your debugging target is the code generated by Flow's AOP Framework then you can start the debugging proxy with `--framework dummy`.

In that case it won't remap from the generated code to your source but "pass through" the debugger steps.
To see what's going on you have to have the generated code in a folder visible to your IDE (in your project).
You can either abstain from `FLOW_PATH_TEMPORARY_BASE` or set it to a path that is in your IDE's project.

Acknowledgments
---------------

Development sponsored by [ttree ltd - neos solution provider](http://ttree.ch).

This project is highly inspired by the PHP based Debug proxy:
https://github.com/sandstorm/debugproxy thanks to the Sandstorm team. The goal
of the Go version of the proxy is to solve the performance issue that the PHP
version has.

We try our best to craft this package with a lots of love, we are open to
sponsoring, support request, ... just contact us.

License
-------

Licensed under MIT, see [LICENSE](LICENSE)
