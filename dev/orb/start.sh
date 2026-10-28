#!/usr/bin/env bash
set -euo pipefail

export GOMAXPROCS=2
export GOFLAGS='-p=2 -buildvcs=false'
sudo bash /opt/orb-worker/docker-start.sh
exec 9>/tmp/orb-dev.lock
flock -n 9 || { echo 'The dev stack is already running' >&2; exit 1; }
sudo sysctl -w fs.inotify.max_user_instances=1024

remove_stale_bridge_rules() {
  local rules
  rules=$(sudo iptables -t raw -S PREROUTING)
  while read -r address bridge; do
    if ! ip link show "$bridge" >/dev/null 2>&1; then
      sudo iptables -t raw -D PREROUTING -d "$address" ! -i "$bridge" -j DROP
    fi
  done < <(awk '
    $3 == "-d" && ($4 == "192.168.49.2/32" || $4 == "192.168.49.3/32") &&
    $5 == "!" && $6 == "-i" && $7 ~ /^br-/ && $8 == "-j" && $9 == "DROP" {print $4, $7}' <<< "$rules")
}

remove_stale_bridge_rules
bash ./dev/start-cluster.sh ./dev/cluster.orb.metal.yaml
remove_stale_bridge_rules
for binary in runsc containerd-shim-runsc-v1 gvisor-bin; do
  docker cp "/usr/local/bin/$binary" "unkey:/usr/local/bin/$binary"
done
docker cp dev/orb/runsc.toml unkey:/etc/containerd/runsc.toml
docker cp dev/orb/runtime.sh unkey:/root/orb-runtime.sh
docker exec unkey bash /root/orb-runtime.sh
kubectl --context unkey apply -f dev/orb/runtimeclass.yaml
exec tilt up --file ./dev/Tiltfile --host 0.0.0.0 --port "${PORT:-10350}" --stream -- --orb-metal=true "$@"
