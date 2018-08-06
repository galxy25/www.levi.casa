include Envfile
export $(shell sed 's/=.*//' Envfile)
PACKAGE_DIR=src
ROOT_PACKAGE=github.com/galxy25/home
GOPATH=$(PWD)
export GOPATH=$(PWD)

.PHONY: install build clean lint test all start run stop restart rere docker_build docker_run docker_tag docker_push docker_pull docker_serve docker_clean

lint :
	echo "Linting"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		go fmt ../...; \
		go vet ../...;

install :
	echo "Installing"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		dep ensure; \
		go get;

build : lint
	echo "Building"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		go install

test : lint build
	echo "Testing"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		go test -v -cover --race -args -project_root=$(PWD); \
		cd communicator; \
		go test -v -timeout 3s -cover --race; \
		cd ../data; \
		go test -v -timeout 3s -cover --race
doc :
	echo "Backgrounding godoc server at http://localhost:2022"
	nohup godoc -http=:2022 >> godoc.out 2>&1 &
	echo "Doc yourself, before you wreck yourself:"
	echo "open http://127.0.0.1:2022/pkg/github.com/galxy25/home/"
	echo "open http://127.0.0.1:2022/pkg/github.com/galxy25/home/internal/?m=all"

stop :
	echo "Stopping web server"
	# Need to double the $$ to get the right
	# substitution value for awk in the below command
	# https://stackoverflow.com/questions/30445218/why-does-awk-not-work-correctly-in-a-makefile
	ps -eax | grep '[b]in/home' | awk '{ print $$1 }' | xargs kill -9

start :
	echo "Running home web server in background"
	echo "Appending output to home.out"
	nohup ./bin/home >> home.out 2>&1 & \
	echo "HOME_PID: $$!"

run : build start

restart : stop start

# i.e. rebuild & restart
rere : stop run
	echo "Rebuilding and restarting web server"

docker_build :
	echo "Building docker image casa from latest source"
	docker build -t casa .

docker_run :
	echo "Running docker image casa:latest"
	docker run -d -p $$CASA_PORT:$$CASA_PORT/tcp --mount type=bind,source="$$(pwd)/data",target=/go/data --env-file Envfile -e AWS_DEFAULT_REGION=$$AWS_DEFAULT_REGION -e AWS_ACCESS_KEY_ID=$$AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY=$$AWS_SECRET_ACCESS_KEY casa:latest

docker_stop :
	echo "Stopping all containers listening on TCP socket 8081"
	docker ps | grep '[8]081/tcp' | awk '{ print $$1 }' | xargs docker kill

docker_restart : docker_stop docker_run
	echo "Restarting web server image"

docker_tag : docker_build
	@echo "Tagging docker image casa for galxy25/www.levi.casa with tag $$VERSION"
	@docker tag casa galxy25/www.levi.casa:$$VERSION

docker_push :
	echo "Pushing all tagged images for galxy25/www.levi.casa"
	docker push galxy25/www.levi.casa

docker_pull :
	docker pull galxy25/www.levi.casa

docker_clean :
	docker rmi -f $$(docker images -qf dangling=true) & \
	docker volume rm $$(docker volume ls -qf dangling=true)

docker_serve :
	docker run -d -p $$CASA_PORT:$$CASA_PORT/tcp -v "$$(pwd)/data":/data --env-file Envfile galxy25/www.levi.casa:latest

clean :
	echo "Cleaning"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		go clean ../...
	rm -rf bin/*
	rm -rf pkg/*
	rm -f home.out
	rm -f godoc.out
	rm -f data/*

all : clean install test doc start
	echo "Installing, linting, building, testing, doc'ing, starting"
