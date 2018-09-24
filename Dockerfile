# Use minimal base golang/linux image
From golang:alpine
# Install curl so we can use it for health checking
# containers of this image
RUN apk add --no-cache curl
# Copy the server binary over
COPY ./bin/linux_home /casa/bin/home
# Copy the client files
COPY ./web /casa/web
# Run the home command by default
# when the container starts.
WORKDIR /casa
ENTRYPOINT /casa/bin/home
#ENTRYPOINT sleep 300
# Document that the service listens on standard web ports
EXPOSE 443 80
# Provide --build-arg HOME_ADDRESS to specify
# the DNS name (required) used to address this
# service.
# See README.md for DNS setup instructions.
ARG HOME_ADDRESS
# Run health checks against the web server's health endpoint
#   /ping
#   every 1 minute
#   timeing out after 3 seconds
HEALTHCHECK --interval=1m --timeout=3s \
  CMD curl -f https://$HOME_ADDRESS/ping || exit 1
