PACKAGE_DIR=src
ROOT_PACKAGE=github.com/galxy25/levis_house
GOPATH=$(PWD)

.PHONY: install build clean lint test all run stop restart docker_build docker_run

lint :
	echo "Linting"
	go fmt ./...
	go vet ./...

install :
	echo "Installing"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		dep ensure

build : install lint
	echo "Building"
	go install ./...

run : build
	echo "Running web server in background"
	echo "Appending output to levis_house.out"
	nohup ./bin/levis_house >> levis_house.out 2>&1 &

test :
	echo "Testing"
	go test ./...

all : install lint build test
	echo "Installing, linting, building, and testing"

stop :
	echo "Stopping web server"
	# Need to double the $ to get the right substitution value
	# substitution value for awk in the below command
	# https://stackoverflow.com/questions/30445218/why-does-awk-not-work-correctly-in-a-makefile
	ps -eax | grep '[b]in/levis_house' | awk '{ print $$1 }' | xargs kill -9

restart : stop run
	echo "Restarted web server"

docker_build : install
	echo "Building docker image casa from latest source"
	docker build -t casa .

docker_run :
	echo "Running docker image casa:latest"
	docker run -d -p 8081:8081/tcp casa:latest

docker_stop :
	echo "Stopping all containers running docker image casa:latest"
	docker ps | grep '[c]asa:latest' | awk '{ print $$1 }' | xargs docker kill

docker_restart : docker_stop docker_run

clean :
	echo "Cleaning"
	go clean ./...
	rm -rf bin/*
	rm -rf pkg/*
	rm levis_house.out
