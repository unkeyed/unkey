#!/usr/bin/env bash
set -euo pipefail

profiles=$(minikube profile list --light -o json)
profile_count=$(jq '[.valid[], .invalid[]] | map(select(.Name == "minikube")) | length' <<< "$profiles")
if jq -e '.invalid[]? | select(.Name == "minikube")' <<< "$profiles" >/dev/null; then
  echo "The minikube profile is invalid. Repair it or explicitly reset it; dev will not delete it." >&2
  exit 1
fi

if [ "$profile_count" -eq 0 ]; then
  minikube_home=${MINIKUBE_HOME:-$HOME}
  if [ "$(basename "$minikube_home")" != .minikube ]; then
    minikube_home="$minikube_home/.minikube"
  fi
  profile_dir="$minikube_home/profiles/minikube"
  if [ -e "$profile_dir" ] || [ -L "$profile_dir" ]; then
    echo "Minikube did not recognize the existing profile at $profile_dir. Repair it before starting dev." >&2
    exit 1
  fi
  exec ctlptl apply -f "$1"
fi

expected_version=$(awk '$1 == "kubernetesVersion:" {print $2}' "$1")
expected_cni=$(sed -n 's/.*"--cni=\([^"]*\)".*/\1/p' "$1")
expected_wait=$(sed -n 's/.*"--wait=\([^"]*\)".*/\1/p' "$1")
if ! jq -e --arg version "$expected_version" --arg cni "$expected_cni" '
  .valid[] | select(.Name == "minikube") | .Config.KubernetesConfig |
  .KubernetesVersion == $version and
  (.CNI == $cni or ($cni == "false" and .CNI == "./dev/k8s/manifests/cilium.yaml")) and
  ($cni != "false" or .NetworkPlugin == "cni")
' <<< "$profiles" >/dev/null; then
  jq -r '.valid[] | select(.Name == "minikube") | .Config.KubernetesConfig |
    "Existing minikube: Kubernetes \(.KubernetesVersion), CNI \(.CNI), network plugin \(.NetworkPlugin)."' <<< "$profiles" >&2
  echo "Required by $1: Kubernetes $expected_version, CNI $expected_cni." >&2
  if [ "$expected_cni" = false ]; then
    echo "Helm-managed Cilium requires network plugin cni." >&2
  fi
  echo "Back up any local data you need, then explicitly reset with mise exec -- minikube delete -p minikube and start dev again. Nothing was deleted." >&2
  exit 1
fi

docker start ctlptl-registry >/dev/null
if ! minikube status -p minikube >/dev/null 2>&1; then
  start_flags=("--cni=$expected_cni" "--wait=$expected_wait" --delete-on-failure=false)
  if [ "$expected_cni" = false ]; then
    start_flags+=(--network-plugin=cni)
  fi
  minikube start -p minikube "${start_flags[@]}"
fi
kubectl config use-context minikube

cluster=$(ctlptl get cluster minikube -o json)
if ! jq -e '.registry == "ctlptl-registry" and .status.localRegistryHosting.host != null' <<< "$cluster" >/dev/null; then
  echo "The existing minikube cluster has no ctlptl registry wiring. Repair it or explicitly reset it; dev will not recreate it." >&2
  exit 1
fi
