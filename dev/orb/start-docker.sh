#!/usr/bin/env bash
set -euo pipefail

if sudo docker info >/dev/null 2>&1; then
  exit 0
fi
if [[ -x /opt/orb-worker/docker-start.sh ]]; then
  sudo bash /opt/orb-worker/docker-start.sh
  exit
fi
if [[ -d /run/systemd/system ]]; then
  sudo systemctl start docker
  exit
fi
# Sandboxes without systemd, such as Depot's managed-agent image.
sudo nohup dockerd </dev/null >/tmp/dockerd.log 2>&1 &
for _ in {1..60}; do
  sudo docker info >/dev/null 2>&1 && exit 0
  sleep 1
done
tail -40 /tmp/dockerd.log >&2
exit 1
