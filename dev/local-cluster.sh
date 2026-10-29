#!/usr/bin/env bash

with_local_cluster() (
  set -euo pipefail

  local manifest=""
  if [ "${1:-}" = --start ]; then
    if [ "$#" -lt 3 ]; then
      echo "Usage: with_local_cluster [--start manifest] command [args...]" >&2
      exit 1
    fi
    manifest=$2
    shift 2
  fi
  if [ "$#" -eq 0 ]; then
    echo "Usage: with_local_cluster [--start manifest] command [args...]" >&2
    exit 1
  fi

  local kubeconfig
  kubeconfig=$(mktemp "${TMPDIR:-/tmp}/unkey-kubeconfig.XXXXXX") || exit
  trap 'rm -f -- "$kubeconfig"' EXIT
  export KUBECONFIG="$kubeconfig"

  if [ -n "$manifest" ]; then
    bash ./dev/start-cluster.sh "$manifest" "$kubeconfig" >&2 || exit
  else
    minikube -p unkey update-context >&2 || exit
  fi
  minikube -p unkey status >&2 || exit

  "$@"
)
