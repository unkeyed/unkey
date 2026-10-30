#!/usr/bin/env bash
set -euo pipefail

export KUBECONFIG="${2:?Use mise run dev to start the cluster with an isolated kubeconfig}"

profiles=$(minikube profile list --light -o json)
profile_count=$(jq '[.valid[], .invalid[]] | map(select(.Name == "unkey")) | length' <<< "$profiles")
if jq -e '.invalid[]? | select(.Name == "unkey")' <<< "$profiles" >/dev/null; then
  echo "The unkey profile is invalid. Repair it or use mise run nuke; dev will not delete it." >&2
  exit 1
fi

registry=$(docker container ls -aq --filter 'name=^/unkey-registry$')
if [ -n "$registry" ] && [ "$(docker inspect "$registry" --format '{{index .Config.Labels "dev.unkey.project"}}')" != unkey ]; then
  echo "Container unkey-registry is not owned by Unkey. Nothing was changed." >&2
  exit 1
fi
volume=$(docker volume ls -q --filter 'name=^unkey$')
if [ -n "$volume" ] && [ "$(docker volume inspect "$volume" --format '{{index .Labels "dev.unkey.project"}}')" != unkey ]; then
  echo "Volume unkey is not owned by Unkey. Nothing was changed." >&2
  exit 1
fi
node=$(docker container ls -aq --filter 'name=^/unkey$')
if [ -n "$node" ] && { [ -z "$volume" ] || [ "$(docker inspect "$node" --format '{{index .Config.Labels "name.minikube.sigs.k8s.io"}}')" != unkey ]; }; then
  echo "Container unkey is not an Unkey cluster node. Nothing was changed." >&2
  exit 1
fi

if [ "$profile_count" -eq 0 ]; then
  minikube_home=${MINIKUBE_HOME:-$HOME}
  if [ "$(basename "$minikube_home")" != .minikube ]; then
    minikube_home="$minikube_home/.minikube"
  fi
  profile_dir="$minikube_home/profiles/unkey"
  if [ -e "$profile_dir" ] || [ -L "$profile_dir" ]; then
    echo "Minikube did not recognize the existing profile at $profile_dir. Repair it before starting dev." >&2
    exit 1
  fi
  docker volume create --name unkey --label dev.unkey.project=unkey \
    --label created_by.minikube.sigs.k8s.io=true --label name.minikube.sigs.k8s.io=unkey >/dev/null
  ctlptl apply -f ./dev/registry.yaml
  exec ctlptl apply -f "$1"
fi

if [ -z "$volume" ]; then
  echo "The unkey profile has no owned node volume. Refusing to adopt it." >&2
  exit 1
fi
expected_version=$(awk '$1 == "kubernetesVersion:" {print $2}' "$1")
expected_cni=$(sed -n 's/.*"--cni=\([^"]*\)".*/\1/p' "$1")
expected_wait=$(sed -n 's/.*"--wait=\([^"]*\)".*/\1/p' "$1")
if ! jq -e --arg version "$expected_version" --arg cni "$expected_cni" '
  .valid[] | select(.Name == "unkey") | .Config.KubernetesConfig |
  .KubernetesVersion == $version and
  (.CNI == $cni or ($cni == "false" and .CNI == "./dev/k8s/manifests/cilium.yaml")) and
  ($cni != "false" or .NetworkPlugin == "cni")
' <<< "$profiles" >/dev/null; then
  jq -r '.valid[] | select(.Name == "unkey") | .Config.KubernetesConfig |
    "Existing minikube: Kubernetes \(.KubernetesVersion), CNI \(.CNI), network plugin \(.NetworkPlugin)."' <<< "$profiles" >&2
  echo "Required by $1: Kubernetes $expected_version, CNI $expected_cni." >&2
  if [ "$expected_cni" = false ]; then
    echo "Helm-managed Cilium requires network plugin cni." >&2
  fi
  echo "Back up any local data you need, then reset with mise run nuke and start dev again. Nothing was deleted." >&2
  exit 1
fi

ctlptl apply -f ./dev/registry.yaml
docker start unkey-registry >/dev/null
if ! minikube -p unkey update-context >/dev/null 2>&1 || ! minikube status -p unkey >/dev/null 2>&1; then
  start_flags=("--cni=$expected_cni" "--wait=$expected_wait" --delete-on-failure=false)
  if [ "$expected_cni" = false ]; then
    start_flags+=(--network-plugin=cni --force)
  fi
  minikube start -p unkey "${start_flags[@]}"
fi

cluster=$(ctlptl get cluster unkey -o json)
if ! jq -e '.registry == "unkey-registry" and .status.localRegistryHosting.host != null' <<< "$cluster" >/dev/null; then
  echo "The existing minikube cluster has no ctlptl registry wiring. Repair it or explicitly reset it; dev will not recreate it." >&2
  exit 1
fi
