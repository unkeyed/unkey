#!/usr/bin/env bash
set -euo pipefail

config_version=$(kubectl -n kube-system patch configmap cilium-config --type merge \
  -p '{"data":{"enable-hubble":"true","hubble-listen-address":":4244"}}' \
  -o jsonpath='{.metadata.resourceVersion}')

kubectl apply -f "$(dirname "$0")/k8s/manifests/hubble.yaml"

kubectl -n kube-system patch daemonset cilium --type merge \
  -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"unkey.dev/cilium-config-version\":\"$config_version\"}}}}}"
kubectl -n kube-system rollout status daemonset cilium --timeout=120s
