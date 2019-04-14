#!/bin/bash
set -x
# Deploys new version of www.levi.casa in a quantum roll-forward manner by default
# $> ./deploy.sh v0.0002 v0.0001
# where the first argument is the new version to deploy
# and the second argument is the old version to roll forward from
# if the old version is not currently running, deployment will not occur
# and any existing running version will be left as is
# (hence a quantum deployment, a discrete step from one level to another)
# to override this behavior and attempt deployment regardless of previous state
# specify force as a third argument, e.g.
# $> ./deploy.sh v0.0002 v0.0001 force
NEW=$1
OLD=$2
FORCE=$3
echo "Pulling image from galxy25/www.levi.casa with tag $NEW"
docker pull galxy25/www.levi.casa:$NEW
if [[ $? -ne 0 ]]; then
    echo "Failed to pull image from galxy25/222.levi.casa with tag $NEW"
    exit 1
fi
echo "stopping any containers running image galxy25/www.levi.casa:$OLD"
Previous=$(docker ps | grep "galxy25/www.levi.casa:$OLD" | awk '{ print $1 }')
if [[ -z "$Previous" ]]; then
    echo "Failed to find any containers running image galxy25/www.levi.casa:$OLD"
    if [[ $FORCE != "force" ]]; then
        exit 1
    else
        echo "Continuing with deployment as force argument was provided"
    fi
fi
docker ps | grep "galxy25/www.levi.casa:$OLD" | awk '{ print $1 }' | xargs docker kill
Previous=$(docker ps | grep "galxy25/www.levi.casa:$OLD" | awk '{ print $1 }')
if [[  ! -z "$Previous" ]]; then
    echo "Failed to stop any containers running image galxy25/www.levi.casa:$OLD"
    if [[ $FORCE != "force" ]]; then
        exit 1
    else
        echo "Continuing with deployment as force argument was provided"
    fi
fi
echo "starting new container with image galxy25/www.levi.casa:$NEW"
if [[ ! -e "./Envfile" ]]; then
    "No Envfile found, deployment can not proceed"
    exit 1
fi
if [[ ! -d "./data" ]]; then
    "No data directory found, deployment can not proceed"
    exit 1
fi
docker run -d --log-driver json-file --log-opt max-size=10m -p 80:80/tcp -p 443:443/tcp -v "$(pwd)/data":/data -v "$(pwd)/tls":/tls --env-file Envfile galxy25/www.levi.casa:$NEW
