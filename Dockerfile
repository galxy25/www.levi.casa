# Use minimal base golang/linux image
From golang:alpine

# Install curl so we can use it for health checking
# containers of this image
RUN apk add --no-cache curl

# Copy the local package files to the container's workspace.
ADD ./src /go/src/
ADD ./static /go/static

# Build the levis_house command inside the container.
# Dependencies are EXPECTED to be installed via go dep from
# running the Makefile target of install in this repo.
RUN go install ./...

# Run the levis_house command by default
# when the container starts.
ENTRYPOINT /go/bin/levis_house

# Document that the service listens on port 8081.
EXPOSE 8081

# Run health checks against the web server's health endpoint
HEALTHCHECK --interval=1m --timeout=3s \
  CMD curl -f http://localhost:8081/ping || exit 1
