# www.levi.casa

Static assets(html, css, js) for my home page.

Golang web server code.

Dockerfile to create an image that can be used to run a container
to serve the static assets using the web server code.

## Pre-requisites
Goland 1.9+ on your system
## Build

### Native Go Binary
Run the following command below:

```
$> make build
```
To install(using `go dep` for dependency resolution), lint, and compile
the go source code into a static binary for the current architecture of your computer.

### Docker Go Binary
```
$> make docker_build
```
To install(using `go dep` for dependency resolution), lint, and compile
the go source code into a static binary for use by the docker build process and image defined in [./Dockerfile]().

## Test

```
$> make test
```

No tests currently written :-(

## Lint

Running the below command:
```
$> make lint
```
Will run the `go fmt` and `go vet` tool on all `.go` files.

## Run

### Naked Process

To run the go web server as a normal process:

```
$> make run
```

Then, access the web site by navigating to http://localhost:8081.

To run a health check against the website, run:

```
$> curl http://localhost:8081/ping
```
Expected healthy response is `pong`

To stop the go web server process
```
$> make stop
```

To stop and start the go web server process

```
$> make restart
```

### Containerized Process

To run the go web server as a docker container(presuming you have followed the step above for how to build the image):

```
$> make docker_run
```

Then, access the web site by navigating to http://localhost:8081.

To run a health check against the website, run:

```
$> curl http://localhost:8081/ping
```
Expected healthy response is `pong`. This is also the health check that the docker agent performs each minute against the running container.

To stop the docker go web server
```
$> make docker_stop
```

To stop and start the go web server process

```
$> make docker_restart
```
