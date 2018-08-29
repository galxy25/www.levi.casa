# Use minimal base golang/linux image
From golang:alpine
# Install curl so we can use it for health checking
# containers of this image
# Install make for running build commands
# Probably a bit of an overkill versus sh'ing things
# Install gcc so we can compile go inside the container
RUN apk add --no-cache curl make gcc g++
# Copy the server src files to the containers go directory
ADD ./src /go/src/
# Copy the client files to the containers web server directory
ADD ./web /go/web
# Copy the services build file to the containers working directory
ADD ./Makefile ./
# Copy app environment file
ADD ./Envfile ./
# Build the home command inside the container.
RUN make build
# Run the home command by default
# when the container starts.
ENTRYPOINT /go/bin/home
# Document that the service listens on port 8081.
EXPOSE 8081
# Run health checks against the web server's health endpoint
#   /ping
#   every 1 minute
#   timeong out after 3 seconds
HEALTHCHECK --interval=1m --timeout=3s \
  CMD curl -f http://localhost:8081/ping || exit 1
