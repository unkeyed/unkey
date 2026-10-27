#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
test "$PWD" -ef "$root" || { echo "Run from the repository root" >&2; exit 1; }
source_dir="$root/svc/heimdall/integration/ciliumplugin"
state_dir="$root/.cache/cilium-integration"
mkdir -p "$state_dir"
export KUBECONFIG="$state_dir/kubeconfig"

mise exec -- minikube start --profile heimdall-cilium --driver=docker \
  --container-runtime=containerd --kubernetes-version=v1.37.0 --cni=false \
  --force --cpus=4 --memory=6144 --wait=apiserver,kubelet --wait-timeout=6m
test "$(mise exec -- kubectl config current-context)" = heimdall-cilium
mise exec -- helm upgrade --install cilium cilium --repo https://helm.cilium.io \
  --version 1.20.2 --namespace kube-system --values "$source_dir/values.yaml" \
  --wait --timeout 6m
mise exec -- kubectl apply -f "$source_dir/fixture.yaml"
mise exec -- kubectl -n heimdall-plugin-test wait --for=condition=Ready pods --all --timeout=180s

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 mise exec -- go build \
  -o "$state_dir/plugin" ./svc/heimdall/integration/ciliumplugin
mise exec -- docker exec heimdall-cilium mkdir -p /opt/heimdall-plugin
if mise exec -- docker exec heimdall-cilium systemctl is-active --quiet heimdall-plugin; then
  mise exec -- docker exec heimdall-cilium systemctl stop heimdall-plugin
fi
if mise exec -- docker exec heimdall-cilium systemctl is-failed --quiet heimdall-plugin; then
  mise exec -- docker exec heimdall-cilium systemctl reset-failed heimdall-plugin
fi
mise exec -- docker cp "$state_dir/plugin" heimdall-cilium:/opt/heimdall-plugin/plugin
mise exec -- docker exec heimdall-cilium systemd-run --unit=heimdall-plugin \
  --property=Type=exec --property=RuntimeDirectory=cilium/plugins/heimdall \
  --property=RuntimeDirectoryMode=0700 \
  /opt/heimdall-plugin/plugin -namespace=heimdall-plugin-test

endpoint=$(mise exec -- kubectl -n heimdall-plugin-test get ciliumendpoint client -o jsonpath='{.status.id}')
target_before=0
if previous=$(mise exec -- docker exec heimdall-cilium /opt/heimdall-plugin/plugin \
  -inspect "/sys/fs/bpf/cilium/endpoints/$endpoint" 2>/dev/null); then
  target_before=$(mise exec -- jq -r '.TargetID' <<<"$previous")
fi
generation=$(date -u +%s%N)
mise exec -- kubectl apply -f - <<EOF
apiVersion: cilium.io/v2alpha1
kind: CiliumDatapathPlugin
metadata:
  name: heimdall
spec:
  attachmentPolicy: BestEffort
  version: "diagnostic-$generation"
EOF

for attempt in {1..60}; do
  endpoint=$(mise exec -- kubectl -n heimdall-plugin-test get ciliumendpoint client -o jsonpath='{.status.id}')
  if current=$(mise exec -- docker exec heimdall-cilium /opt/heimdall-plugin/plugin \
    -inspect "/sys/fs/bpf/cilium/endpoints/$endpoint"); then
    if test "$(mise exec -- jq -r '.TargetID' <<<"$current")" != "$target_before"; then
      printf '%s\n' "$current"
      exit 0
    fi
  fi
  sleep 1
done
echo "Plugin attachment did not become observable" >&2
exit 1
