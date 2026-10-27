#!/usr/bin/env bash
set -euo pipefail

runsc --version
if ! grep -q 'runtimes.runsc' /etc/containerd/config.toml; then
  cat >> /etc/containerd/config.toml <<'EOF'

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
runtime_type = "io.containerd.runsc.v1"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc.options]
TypeUrl = "io.containerd.runsc.v1.options"
ConfigPath = "/etc/containerd/runsc.toml"
EOF
fi
containerd --config /etc/containerd/config.toml config dump >/dev/null
systemctl restart containerd
