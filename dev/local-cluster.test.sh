#!/usr/bin/env bash
set -euo pipefail

root=$(mktemp -d)
trap 'rm -rf -- "$root"' EXIT
export HOME="$root/home"
export TMPDIR="$root/tmp"
export TEST_STATE=running
export TEST_ROOT="$root"
mkdir -p "$HOME/.kube" "$TMPDIR" "$root/bin"
export KUBECONFIG="$HOME/.kube/config"
kubectl config set-cluster production --server=https://production.invalid >/dev/null
kubectl config set-credentials production --exec-command=must-not-run >/dev/null
kubectl config set-context production --cluster=production --user=production >/dev/null
kubectl config set-context unkey --cluster=production --user=production >/dev/null
kubectl config use-context production >/dev/null
cp "$KUBECONFIG" "$root/original"

cat > "$root/bin/minikube" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_ROOT/minikube-calls"
if [ "$*" = 'profile list --light -o json' ]; then
  if [ "$TEST_STATE" = missing ]; then
    jq -n '{valid: [], invalid: []}'
  else
    jq -n '{valid: [{Name: "unkey", Config: {KubernetesConfig: {KubernetesVersion: "v1.37.0", CNI: "false", NetworkPlugin: "cni"}}}], invalid: []}'
  fi
  exit
fi
if [ "$1" = -p ]; then
  [ "$2" = unkey ]
  shift 2
fi
case "$1" in
  update-context)
    [ "$TEST_STATE" != stopped ] || exit 7
    ;;
  start)
    : > "$TEST_ROOT/started"
    ;;
  status)
    [ "$TEST_STATE" != unhealthy ] || exit 7
    server=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
    [ "$server" = https://127.0.0.1:6443 ] || exit 4
    echo 'host: Running'
    exit
    ;;
  *) exit 99 ;;
esac
kubectl config set-cluster unkey --server=https://127.0.0.1:6443 --certificate-authority="$HOME/.minikube/ca.crt"
kubectl config set-credentials unkey --client-certificate="$HOME/.minikube/profiles/unkey/client.crt" --client-key="$HOME/.minikube/profiles/unkey/client.key"
kubectl config set-context unkey --cluster=unkey --user=unkey
kubectl config use-context unkey
SH

cat > "$root/bin/docker" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  'container ls '*) ;;
  'volume ls '*) [ "$TEST_STATE" = missing ] || echo unkey ;;
  'volume inspect '*|'volume create '*|'start '*) echo unkey ;;
  *) exit 99 ;;
esac
SH

cat > "$root/bin/ctlptl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  'apply -f ./dev/registry.yaml') ;;
  'apply -f ./dev/cluster.yaml') minikube -p unkey start ;;
  'get cluster unkey -o json')
    jq -n '{registry: "unkey-registry", status: {localRegistryHosting: {host: "localhost:5000"}}}'
    ;;
  *) exit 99 ;;
esac
SH
chmod +x "$root/bin/"*
export PATH="$root/bin:$PATH"

cat > "$root/probe" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[ "$KUBECONFIG" != "$HOME/.kube/config" ]
[ "$(ls -l "$KUBECONFIG" | awk '{print $1}')" = -rw------- ]
printf '%s\n' "$KUBECONFIG" >> "$TEST_ROOT/paths"
kubectl config view -o json | jq -e '
  .["current-context"] == "unkey" and
  (.contexts | length) == 1 and
  (.clusters | length) == 1 and
  .clusters[0].cluster.server == "https://127.0.0.1:6443" and
  (.users | length) == 1 and
  .users[0].user.exec == null and
  (.users[0].user["client-certificate"] | endswith("/unkey/client.crt"))
' >/dev/null
if kubectl --context production config view --minify >/dev/null 2>&1; then
  echo 'Production context is still accessible' >&2
  exit 1
fi
printf '%s\n' "$*"
SH

output=$(bash .mise/tasks/local-cluster bash "$root/probe" 'argument with spaces' 2>"$root/log")
[ "$output" = 'argument with spaces' ]
cmp "$KUBECONFIG" "$root/original"
echo 'PASS: command sees only local certificates; caller kubeconfig and stdout are unchanged'

for state in running missing stopped; do
  export TEST_STATE="$state"
  bash .mise/tasks/local-cluster --start ./dev/cluster.yaml bash "$root/probe" > /dev/null 2>"$root/log"
  cmp "$KUBECONFIG" "$root/original"
  if [ "$state" = running ]; then
    [ ! -f "$root/started" ]
  else
    [ -f "$root/started" ]
  fi
  echo "PASS: $state cluster startup uses isolated kubeconfig"
done

export TEST_STATE=unhealthy
if bash .mise/tasks/local-cluster touch "$root/unsafe-command" 2>"$root/log"; then
  echo 'Unhealthy cluster was accepted' >&2
  exit 1
else
  [ "$?" -eq 7 ]
fi
[ ! -e "$root/unsafe-command" ]
echo 'PASS: failed health check prevents command execution'

export TEST_STATE=stopped
if bash .mise/tasks/local-cluster touch "$root/unsafe-command" 2>"$root/log"; then
  echo 'Stopped cluster was accepted' >&2
  exit 1
else
  [ "$?" -eq 7 ]
fi
[ ! -e "$root/unsafe-command" ]
echo 'PASS: missing endpoint prevents command execution'

export TEST_STATE=running
if bash .mise/tasks/local-cluster bash -c 'exit 23' 2>"$root/log"; then
  echo 'Command failure was lost' >&2
  exit 1
else
  [ "$?" -eq 23 ]
fi
[ -z "$(find "$TMPDIR" -name 'unkey-kubeconfig.*' -print)" ]
[ "$(sort -u "$root/paths" | wc -l | tr -d ' ')" -eq 4 ]
cmp "$KUBECONFIG" "$root/original"
echo 'PASS: separate kubeconfigs are removed on success and failure; exit status is preserved'
