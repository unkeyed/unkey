#!/usr/bin/env bash
set -euo pipefail

# Depot restores HOME, but not the disposable Docker disk.
if [[ $(docker info --format '{{.DockerRootDir}}') != /var/lib/orb-docker ]]; then
  exit 0
fi
volume=$(docker volume ls -q --filter 'name=^buildx_buildkit_unkey0_state$')
container=$(docker ps -aq --filter 'name=^/buildx_buildkit_unkey0$')
if [[ -z $volume && -z $container ]] && builder=$(docker buildx inspect unkey 2>/dev/null); then
  endpoint=${DOCKER_HOST:-$(docker context show)}
  host=${DOCKER_HOST:-$(docker context inspect --format '{{.Endpoints.docker.Host}}')}
  builder_endpoint=$(awk '$1 == "Endpoint:" {print $2}' <<< "$builder")
  if [[ $(awk '$1 == "Driver:" {print $2}' <<< "$builder") == docker-container &&
        $(awk '$1 == "Name:" {print $2}' <<< "$builder") == $'unkey\nunkey0' &&
        $(awk '$1 == "Status:" {print $2}' <<< "$builder") == inactive &&
        ( $builder_endpoint == "$endpoint" || $builder_endpoint == "$host" ) ]]; then
    docker buildx rm --keep-daemon --keep-state unkey
  fi
fi
"$HOME/.local/bin/mise" exec -- env KUBECONFIG=/dev/null minikube delete -p unkey
