# www.levi.casa

Levi Schoen's digital home, consisting of front end assets(html, css, js) and backend web server and services code.

# Runtime requirements

* If you want to run code as a bare binary:
    * Linux/macOS
    * Golang 1.11+
* If you want to run code as a docker image
    * docker

## Build

### Native Go Binary
Run the following command below:

```
$> make build
```

To install(using `go dep` for dependency resolution), lint, and compile
the go source code into a static binary in [./bin](./bin) for the current architecture of your computer.

### Docker Image

```
$> make docker_build
```

To build a runnable docker image of www.levi.casa

## Test

```
$> make test
```

Runs unit and Integration tests.
Integration tests require that the following values
are set in the shell from which tests are run:

* AWS_DEFAULT_REGION
* AWS_ACCESS_KEY_ID
* AWS_SECRET_ACCESS_KEY
* TWILIO_ACCOUNT_SID
* TWILIO_AUTH_TOKEN

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

To stop, rebuild, and start the go web server process

```
$> make rere
```

### Containerized Process

To run the go web server as a docker container(presuming you have followed previous step above for how to build the image):

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

## Clean

## Deploy

To deploy tagged docker image v2 on an environment where docker image v2 is running, run:

```
$> ./deploy.sh v2 v1
```

To force a deployment of v2 even if v1 is not running:

```
$> ./deploy.sh v2 v1 force
```

Envfile must have valid values for:

* AWS_DEFAULT_REGION
* AWS_ACCESS_KEY_ID
* AWS_SECRET_ACCESS_KEY
* TWILIO_ACCOUNT_SID
* TWILIO_AUTH_TOKEN

along with valid/the same values as located in this repo's Envfile
for deployed app to be functional.
