# Cursor Cloud helpers. Amp orbs do not source this file.
# shellcheck shell=bash

wait_for_docker() {
  local attempts="${1:-60}"
  local attempt
  for attempt in $(seq 1 "$attempts"); do
    grant_docker_sock
    if docker info >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

install_fuse_overlayfs() {
  if command -v fuse-overlayfs >/dev/null 2>&1 && command -v setfacl >/dev/null 2>&1; then
    return 0
  fi
  export DEBIAN_FRONTEND=noninteractive
  sudo apt-get update
  sudo apt-get install -y -o Dpkg::Options::=--force-confold fuse3 fuse-overlayfs acl
}

# Cursor Cloud rejects nested overlayfs. Write the host Docker config before
# the first dockerd start; overlay2 never becomes ready in these VMs.
write_fuse_overlayfs_daemon_json() {
  sudo mkdir -p /etc/docker
  if [ -f /etc/docker/daemon.json ]; then
    sudo python3 - <<'PY'
import json
from pathlib import Path

path = Path("/etc/docker/daemon.json")
data = json.loads(path.read_text() or "{}")
if data.get("storage-driver") == "fuse-overlayfs":
    raise SystemExit(0)
data["storage-driver"] = "fuse-overlayfs"
path.write_text(json.dumps(data, indent=2) + "\n")
PY
  else
    echo '{"storage-driver":"fuse-overlayfs"}' | sudo tee /etc/docker/daemon.json >/dev/null
  fi
}

print_dockerd_log() {
  echo "docker diagnostics:" >&2
  pgrep -a dockerd >&2 || echo "no dockerd process" >&2
  ls -l /var/run/docker.sock >&2 || echo "no docker.sock" >&2
  if [ -f /tmp/unkey-dockerd.log ]; then
    echo "dockerd log:" >&2
    sudo tail -n 80 /tmp/unkey-dockerd.log >&2 || true
  fi
}

stop_stale_dockerd() {
  local pid
  for pid in $(pgrep -x dockerd || true); do
    sudo kill "$pid" || true
  done
  sleep 1
  for pid in $(pgrep -x dockerd || true); do
    sudo kill -9 "$pid" || true
  done
  sleep 1
}

# Cursor Cloud VMs have no usable systemd. Do not call systemctl or service.
start_dockerd_directly() {
  echo "Starting dockerd..."
  stop_stale_dockerd
  sudo mkdir -p /var/run
  sudo dockerd \
    --host=unix:///var/run/docker.sock \
    --storage-driver=fuse-overlayfs \
    >/tmp/unkey-dockerd.log 2>&1 &
}

start_docker() {
  if docker info >/dev/null 2>&1; then
    return 0
  fi

  install_fuse_overlayfs
  write_fuse_overlayfs_daemon_json
  start_dockerd_directly
  if wait_for_docker; then
    return 0
  fi

  echo "Docker did not become ready" >&2
  print_dockerd_log
  return 1
}

grant_docker_sock() {
  sudo usermod -aG docker "$(id -un)" || true
  if [ ! -S /var/run/docker.sock ]; then
    return 0
  fi
  if command -v setfacl >/dev/null 2>&1; then
    sudo setfacl -m "u:$(id -un):rw" /var/run/docker.sock || sudo chmod 666 /var/run/docker.sock
  else
    sudo chmod 666 /var/run/docker.sock
  fi
}

raise_inotify_quota() {
  printf '1024\n' | sudo tee /proc/sys/fs/inotify/max_user_instances >/dev/null
}

install_docker_if_needed() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    return 0
  fi

  . /etc/os-release
  local docker_os="$ID"
  case "$docker_os" in
    debian | ubuntu) ;;
    *)
      echo "Unsupported OS for Docker install: $docker_os" >&2
      return 1
      ;;
  esac

  sudo apt-get update
  sudo apt-get install -y ca-certificates curl
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL "https://download.docker.com/linux/${docker_os}/gpg" -o /etc/apt/keyrings/docker.asc
  sudo chmod a+r /etc/apt/keyrings/docker.asc

  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${docker_os} ${VERSION_CODENAME} stable" |
    sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

  sudo apt-get update
  sudo apt-get install -y \
    containerd.io \
    docker-buildx-plugin \
    docker-ce \
    docker-ce-cli \
    docker-compose-plugin
}

ensure_docker_storage() {
  install_fuse_overlayfs
  write_fuse_overlayfs_daemon_json

  local current
  current="$(docker info --format '{{.Driver}}' 2>/dev/null || true)"
  if [ "$current" = "fuse-overlayfs" ]; then
    return 0
  fi

  start_dockerd_directly
  if wait_for_docker; then
    grant_docker_sock
    return 0
  fi

  echo "Docker did not become ready after switching to fuse-overlayfs" >&2
  print_dockerd_log
  return 1
}

# Force native snapshotter and at most one gvisor runsc table. minikube start
# re-enables gvisor and otherwise appends a duplicate table.
fix_containerd_config() {
  docker inspect minikube >/dev/null 2>&1 || return 1
  [ "$(docker inspect -f '{{.State.Running}}' minikube 2>/dev/null)" = "true" ] || return 1

  docker exec -i minikube sh -s <<'EOS'
cfg=/etc/containerd/config.toml
[ -f "$cfg" ] || exit 0
tmp=$(mktemp)
awk '
  { line = $0 }
  line ~ /snapshotter[[:space:]]*=/ {
    sub(/snapshotter[[:space:]]*=[[:space:]]*"[^"]*"/, "snapshotter = \"native\"", line)
  }
  line == "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runsc]" {
    if (++runsc > 1) { drop = 1; next }
  }
  drop {
    if (line ~ /^\[/ && line != "[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runsc]") {
      drop = 0
    } else {
      next
    }
  }
  { print line }
' "$cfg" >"$tmp"
if cmp -s "$cfg" "$tmp"; then
  rm -f "$tmp"
  exit 0
fi
mv "$tmp" "$cfg"
pid=$(pidof containerd | awk '{print $1}')
[ -n "$pid" ] && kill "$pid"
EOS
}

# Images pinned by minikube 1.38.1 addons. ImagePullBackOff import below
# covers version drift when minikube is bumped.
minikube_addon_images() {
  cat <<'EOF'
registry.k8s.io/metrics-server/metrics-server:v0.8.1@sha256:b2d2efaf5ac3b366ed0f839d2412a2c4279d4fc2a2a733f12c52133faed36c41
registry.k8s.io/minikube/gvisor:v0.0.4@sha256:0f389d92114b6342bcdb971fc8e89e9d60683d49aa5e31b89d744ec420196fd9
EOF
}

import_image_into_minikube() {
  local spec="$1"
  echo "Importing $spec into minikube..."
  docker pull "$spec"
  local id
  id="$(docker image inspect --format '{{.Id}}' "$spec")"
  docker save "$id" | docker exec -i minikube ctr -n k8s.io images import -
}

list_unpulled_pod_images() {
  local mise="$1"
  "$mise" exec -- kubectl get pods -A -o json 2>/dev/null | python3 -c '
import json, sys

data = json.load(sys.stdin)
images = set()
waiting = {"ImagePullBackOff", "ErrImagePull"}
for item in data.get("items", []):
    status = item.get("status") or {}
    for key in ("containerStatuses", "initContainerStatuses"):
        for cs in status.get(key) or []:
            state = ((cs.get("state") or {}).get("waiting") or {})
            if state.get("reason") in waiting:
                image = cs.get("image") or ""
                if image:
                    images.add(image)
print("\n".join(sorted(images)))
'
}

import_minikube_images() {
  local mise="$1"
  docker inspect minikube >/dev/null 2>&1 || return 0

  local image
  while IFS= read -r image; do
    [ -z "$image" ] && continue
    import_image_into_minikube "$image" || true
  done < <(minikube_addon_images)

  sleep 8

  local attempt pending
  for attempt in $(seq 1 12); do
    pending="$(list_unpulled_pod_images "$mise" || true)"
    if [ -z "$pending" ] && "$mise" exec -- kubectl get nodes >/dev/null 2>&1; then
      return 0
    fi
    while IFS= read -r image; do
      [ -z "$image" ] && continue
      import_image_into_minikube "$image" || true
    done <<<"$pending"
    sleep 5
  done
}

configure_kube_proxy() {
  local mise="${1:-$HOME/.local/bin/mise}"
  "$mise" exec -- kubectl -n kube-system get cm kube-proxy >/dev/null 2>&1 || return 0

  python3 - "$mise" <<'PY'
import json, subprocess, sys

mise = sys.argv[1]
raw = subprocess.check_output(
    [mise, "exec", "--", "kubectl", "-n", "kube-system", "get", "cm", "kube-proxy", "-o", "json"]
)
cm = json.loads(raw)
conf = cm["data"]["config.conf"]
if "\nmode: nftables" in conf or conf.startswith("mode: nftables"):
    sys.exit(0)
lines = []
found = False
for line in conf.splitlines():
    if line.startswith("mode:"):
        lines.append("mode: nftables")
        found = True
    else:
        lines.append(line)
if not found:
    lines.append("mode: nftables")
cm["data"]["config.conf"] = "\n".join(lines) + "\n"
subprocess.run(
    [mise, "exec", "--", "kubectl", "-n", "kube-system", "patch", "cm", "kube-proxy",
     "--type", "merge", "-p", json.dumps({"data": {"config.conf": cm["data"]["config.conf"]}})],
    check=True,
)
subprocess.run(
    [mise, "exec", "--", "kubectl", "-n", "kube-system", "delete", "pod", "-l", "k8s-app=kube-proxy"],
    check=False,
)
print("kube-proxy switched to nftables mode")
PY
}

configure_insecure_registry() {
  docker inspect minikube >/dev/null 2>&1 || return 0
  docker inspect ctlptl-registry >/dev/null 2>&1 || return 0

  local ip first_ip=""
  for ip in $(docker inspect ctlptl-registry --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}'); do
    [ -n "$ip" ] || continue
    [ -n "$first_ip" ] || first_ip="$ip"
    docker exec minikube sh -c "
      mkdir -p /etc/containerd/certs.d/${ip}:5000
      printf '%s\n' '[host.\"http://${ip}:5000\"]' > /etc/containerd/certs.d/${ip}:5000/hosts.toml
    "
  done

  # kubelet pulls on the node, so cluster DNS for the ctlptl-registry
  # Service never runs. Bridge-network minikube also cannot resolve that
  # hostname through Docker DNS. Point it at the registry's docker0 IP.
  if [ -n "$first_ip" ]; then
    docker exec minikube sh -c "
      grep -v '[[:space:]]ctlptl-registry\$' /etc/hosts > /tmp/hosts.cursor
      printf '%s ctlptl-registry\n' '${first_ip}' >> /tmp/hosts.cursor
      cat /tmp/hosts.cursor > /etc/hosts
      mkdir -p /etc/containerd/certs.d/ctlptl-registry:5000
      printf '%s\n' '[host.\"http://${first_ip}:5000\"]' > /etc/containerd/certs.d/ctlptl-registry:5000/hosts.toml
    "
  fi
}

finish_cluster() {
  local mise="${1:-$HOME/.local/bin/mise}"
  fix_containerd_config || true
  configure_insecure_registry || true
  configure_kube_proxy "$mise" || true
  import_minikube_images "$mise"
}

_apply_cluster_locked() {
  local mise="${1:-$HOME/.local/bin/mise}"
  local patch_pid=""

  if [ "$(docker inspect -f '{{.State.Running}}' minikube 2>/dev/null)" = "true" ] &&
    "$mise" exec -- kubectl get nodes >/dev/null 2>&1; then
    echo "minikube already reachable"
    finish_cluster "$mise"
    return 0
  fi

  (
    while true; do
      fix_containerd_config || true
      sleep 2
    done
  ) &
  patch_pid=$!

  if ! "$mise" exec -- ctlptl apply -f dev/cluster.cursor.yaml; then
    kill "$patch_pid" 2>/dev/null || true
    wait "$patch_pid" 2>/dev/null || true
    return 1
  fi

  kill "$patch_pid" 2>/dev/null || true
  wait "$patch_pid" 2>/dev/null || true
  finish_cluster "$mise"
}

apply_cluster() {
  local mise="${1:-$HOME/.local/bin/mise}"
  (
    flock 9
    _apply_cluster_locked "$mise"
  ) 9>/tmp/unkey-cursor-cluster.lock
}
