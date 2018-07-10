PACKAGE_DIR=src
ROOT_PACKAGE=github.com/galxy25/levishouse
GOPATH=$(PWD)
export GOPATH=$(PWD)

.PHONY: install build clean lint test all run stop restart rere docker_build docker_run docker_tag docker_push docker_pull docker_serve docker_clean

lint :
	echo "Linting"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		go fmt; \
		go vet *.go

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
		go test -v -cover --race -args -test_bin_dir=$(PWD)

doc :
	echo "Backgrounding godoc server at http://localhost:2022"
	nohup godoc -http=:2022 >> godoc.out 2>&1 &
	echo "Doc yourself, before you wreck yourself:"
	echo "open http://127.0.0.1:2022/pkg/github.com/galxy25/levishouse/"
	echo "open http://127.0.0.1:2022/pkg/github.com/galxy25/levishouse/internal/?m=all"

run :
	echo "Running web server in background"
	echo "Appending output to levis_house.out"
	DESIRED_CONNECTIONS_FILEPATH="$$(pwd)/data/desired_connections.txt" \
	CURRENT_CONNECTIONS_FILEPATH="$$(pwd)/data/current_connections.txt" \
	nohup ./bin/levishouse >> levis_house.out 2>&1 & \
	echo "LEVISHOUSE_PID: $$!"

all : clean install test doc run
	echo "Installing, linting, building, testing, doc'ing, running"

stop :
	echo "Stopping web server"
	# Need to double the $$ to get the right
	# substitution value for awk in the below command
	# https://stackoverflow.com/questions/30445218/why-does-awk-not-work-correctly-in-a-makefile
	ps -eax | grep '[b]in/levishouse' | awk '{ print $$1 }' | xargs kill -9

restart : stop run
	echo "Restarted web server"

# restart & rebuild
rere : stop build run
	echo "Rebuilding and restarting web server"

docker_build :
	echo "Building docker image casa from latest source"
	docker build -t casa .

docker_run :
	echo "Running docker image casa:latest"
	docker run -d -p 8081:8081/tcp --mount type=bind,source="$$(pwd)/data",target=/data --env-file casa.env casa:latest

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
	docker run -d -p 8081:8081/tcp -v "$$(pwd)":/data --env-file casa.env galxy25/www.levi.casa:latest

clean :
	echo "Cleaning"
	cd $(PACKAGE_DIR)/$(ROOT_PACKAGE); \
		go clean
	rm -rf bin/*
	rm -rf pkg/*
	rm -f levis_house.out
	rm -f godoc.out
	rm -f data/desired_connections.txt
	rm -f data/current_connections.txt
