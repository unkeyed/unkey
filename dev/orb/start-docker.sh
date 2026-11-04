#!/usr/bin/env bash
set -euo pipefail

if sudo -n docker info >/dev/null 2>&1; then
  exit 0
fi
if [[ -x /opt/orb-worker/docker-start.sh ]]; then
  sudo -n bash /opt/orb-worker/docker-start.sh
  exit
fi
if [[ -d /run/systemd/system ]]; then
  sudo -n systemctl start docker
  exit
fi
# Sandboxes without systemd, such as Depot's managed-agent image.
sudo -n sh -c 'umask 077; nohup dockerd </dev/null >>/var/log/orb-dockerd.log 2>&1 &'
for _ in {1..60}; do
  sudo -n docker info >/dev/null 2>&1 && exit 0
  sleep 1
done
sudo -n tail -40 /var/log/orb-dockerd.log >&2
exit 1
