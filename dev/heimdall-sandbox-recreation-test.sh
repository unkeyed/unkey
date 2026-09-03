#!/usr/bin/env bash
#
# End-to-end check that heimdall keeps metering a pod after its gVisor
# sandbox is recreated under the same pod UID.
#
# Runs against the local dev cluster from `mise run dev` (minikube +
# containerd + cilium + gvisor addon). Heimdall must be running in the
# `unkey` namespace from the code under test; Tilt rebuilds it from source.
#
# What it does:
#   1. Creates a krane-labelled busybox pod with runtimeClassName=gvisor and a
#      64Mi memory limit. The pod sends a little egress every second.
#   2. Waits until heimdall writes attached checkpoints for it (baseline).
#   3. Kills the sandbox with `crictl stopp` (StopPodSandbox). That is the
#      containerd-level event production hits when gVisor OOM-kills a pod: the
#      netns file is removed, the stopped sandbox record stays until GC, and
#      kubelet builds a new sandbox with a new netns under the same pod UID.
#      TRIGGER=oom fills memory inside the container instead. Whether that
#      takes the whole sandbox down depends on the runsc build the minikube
#      addon downloaded (release/latest); newer builds may kill only the
#      process and leave the sandbox up, in which case the script reports
#      "container restarted, sandbox kept" and fails.
#   4. Waits for the new sandbox, then for checkpoints written after it.
#   5. PASS if the last checkpoints have attributes.network_attached=true and
#      the egress counter grows across them. FAIL otherwise.
#
# Before the fix (#7254) step 5 fails: every checkpoint after the first
# recreation has network_attached=false until heimdall restarts.
#
# Env:
#   TRIGGER=stopp|oom   how to kill the sandbox (default stopp)
#   KEEP=1              leave the repro namespace in place after the run
#   TIMEOUT=<seconds>   per-phase wait budget (default 180)

set -euo pipefail

TRIGGER="${TRIGGER:-stopp}"
KEEP="${KEEP:-0}"
TIMEOUT="${TIMEOUT:-180}"

NS=heimdall-repro
POD=sandbox-recreation
WORKSPACE=ws_heimdall_repro
HEIMDALL_NS=unkey
CH_USER=default
CH_PASSWORD=password
# heimdall checkpoints every 5s and flushes to ClickHouse every 5s. The first
# tick or two after a new sandbox may legitimately be unattached (sandbox not
# ready, then the one-tick ErrNotAttached gap), so wait for a few extra rows
# and judge only the last VERDICT_ROWS.
VERDICT_ROWS=4
SETTLE_ROWS=2

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
ok() { printf '\033[1;32mPASS\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || fail "missing tool: $1"; }
need kubectl
need minikube

now_ms() { echo $(( $(date +%s) * 1000 )); }

ch() {
  kubectl -n "$HEIMDALL_NS" exec deploy/clickhouse -- \
    clickhouse-client --user "$CH_USER" --password "$CH_PASSWORD" --format TSV --query "$1"
}

# Newest containerd sandbox id for the pod. crictl lists stopped sandboxes
# too until kubelet GC removes them, which is exactly the window the bug
# lives in.
newest_sandbox() {
  minikube ssh -- "sudo crictl pods --namespace $NS --name $POD -q" 2>/dev/null | tr -d '\r' | head -n1
}

pod_field() { kubectl -n "$NS" get pod "$POD" -o jsonpath="$1" 2>/dev/null || true; }

# wait_for <description> <predicate function>; polls until it returns 0.
# Returns 1 on timeout, which ends the script under set -e unless the caller
# checks it.
wait_for() {
  local desc=$1 pred=$2 start=$SECONDS
  until "$pred"; do
    if (( SECONDS - start > TIMEOUT )); then
      printf '\033[1;31mFAIL\033[0m timed out after %ss waiting for: %s\n' "$TIMEOUT" "$desc" >&2
      return 1
    fi
    sleep 2
  done
}

cleanup() {
  if [[ "$KEEP" == "1" ]]; then
    log "KEEP=1, leaving namespace $NS in place"
    return
  fi
  kubectl delete namespace "$NS" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup EXIT

# --- preconditions ----------------------------------------------------------

log "checking cluster"
kubectl get runtimeclass gvisor >/dev/null 2>&1 \
  || fail "RuntimeClass gvisor missing; run mise run dev (dev/cluster.yaml enables the gvisor addon)"
kubectl -n "$HEIMDALL_NS" rollout status ds/heimdall --timeout=60s >/dev/null \
  || fail "heimdall DaemonSet not ready in $HEIMDALL_NS"
kubectl -n "$HEIMDALL_NS" rollout status deploy/clickhouse --timeout=60s >/dev/null \
  || fail "clickhouse not ready in $HEIMDALL_NS"
ch "SELECT 1" >/dev/null || fail "cannot query clickhouse"
HEIMDALL_POD=$(kubectl -n "$HEIMDALL_NS" get pod -l app=heimdall -o jsonpath='{.items[0].metadata.name}')
log "heimdall pod: $HEIMDALL_POD"

# --- create the pod ---------------------------------------------------------

log "creating $NS/$POD (gvisor, 64Mi limit, krane labels)"
kubectl delete namespace "$NS" --ignore-not-found --wait=true >/dev/null
kubectl create namespace "$NS" >/dev/null
kubectl apply -f - >/dev/null <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $POD
  namespace: $NS
  labels:
    app.kubernetes.io/component: deployment
    app.kubernetes.io/managed-by: krane
    unkey.com/workspace.id: $WORKSPACE
    unkey.com/project.id: proj_heimdall_repro
    unkey.com/app.id: app_heimdall_repro
    unkey.com/environment.id: env_heimdall_repro
    unkey.com/deployment.id: dep_heimdall_repro
spec:
  runtimeClassName: gvisor
  restartPolicy: Always
  containers:
    - name: app
      image: busybox:1.36
      # Steady egress so counter growth proves the new attach measures the
      # new netns. dev/k8s/manifests/cilium-policies.yaml lets krane-labelled
      # pods reach DNS and krane:8070 only; both are used here.
      command:
        - sh
        - -c
        - |
          while true; do
            nslookup kubernetes.default.svc.cluster.local >/dev/null 2>&1 || true
            wget -qO /dev/null -T 2 http://krane.unkey.svc.cluster.local:8070/ 2>/dev/null || true
            sleep 1
          done
      resources:
        requests:
          cpu: 10m
          memory: 32Mi
        limits:
          memory: 64Mi
EOF

pod_running() { [[ $(pod_field '{.status.phase}') == Running ]]; }
sandbox_visible() { [[ -n $(newest_sandbox) ]]; }

wait_for "pod Running" pod_running
wait_for "containerd sandbox visible" sandbox_visible
POD_UID=$(pod_field '{.metadata.uid}')
IP_BEFORE=$(pod_field '{.status.podIP}')
SANDBOX_BEFORE=$(newest_sandbox)
log "pod uid=$POD_UID ip=$IP_BEFORE sandbox=${SANDBOX_BEFORE:0:12}"

# --- baseline: heimdall attached to the first sandbox -----------------------

rows_since() {
  ch "SELECT count() FROM instance_checkpoints_v1
      WHERE workspace_id = '$WORKSPACE' AND pod_uid = '$POD_UID' AND ts >= $1"
}

attached_rows_since() {
  ch "SELECT count() FROM instance_checkpoints_v1
      WHERE workspace_id = '$WORKSPACE' AND pod_uid = '$POD_UID' AND ts >= $1
        AND ifNull(attributes.network_attached::Nullable(Bool), false)"
}

T0=$(now_ms)
baseline_attached() { (( $(attached_rows_since "$T0") >= 2 )); }
log "waiting for baseline attached checkpoints"
wait_for "2 attached checkpoints before recreation" baseline_attached
ok "baseline: heimdall meters the first sandbox"

# --- kill the sandbox --------------------------------------------------------

RESTARTS_BEFORE=$(pod_field '{.status.containerStatuses[0].restartCount}')
T_KILL=$(now_ms)

case "$TRIGGER" in
  stopp)
    log "trigger: crictl stopp ${SANDBOX_BEFORE:0:12}"
    minikube ssh -- "sudo crictl stopp $SANDBOX_BEFORE" >/dev/null
    ;;
  oom)
    log "trigger: OOM inside the container"
    # Doubling a string allocates exponentially: past 64Mi within ~27
    # iterations, well under a second. The exec dies with the process.
    kubectl -n "$NS" exec "$POD" -- awk 'BEGIN { s = "x"; while (1) s = s s }' >/dev/null 2>&1 || true
    ;;
  *)
    fail "unknown TRIGGER=$TRIGGER (want oom or stopp)"
    ;;
esac

sandbox_recreated() {
  local newest
  newest=$(newest_sandbox)
  [[ -n "$newest" && "$newest" != "$SANDBOX_BEFORE" ]] \
    && pod_running \
    && [[ $(pod_field '{.metadata.uid}') == "$POD_UID" ]]
}

dump_state() {
  echo "--- diagnostics ---" >&2
  kubectl -n "$NS" get pod "$POD" -o wide 2>&1 | sed 's/^/  /' >&2 || true
  echo "  restartCount before=$RESTARTS_BEFORE now=$(pod_field '{.status.containerStatuses[0].restartCount}')" >&2
  echo "  containerd sandboxes for the pod (newest first):" >&2
  minikube ssh -- "sudo crictl pods --namespace $NS --name $POD" 2>/dev/null | tr -d '\r' | sed 's/^/    /' >&2 || true
  echo "  recent pod events:" >&2
  kubectl -n "$NS" get events --field-selector "involvedObject.name=$POD" --sort-by=.lastTimestamp 2>/dev/null | tail -n 8 | sed 's/^/    /' >&2 || true
  if [[ "$TRIGGER" == oom && $(pod_field '{.status.containerStatuses[0].restartCount}') != "$RESTARTS_BEFORE" ]]; then
    echo "  container restarted, sandbox kept: this runsc build kills the process, not the sandbox. Use TRIGGER=stopp." >&2
  fi
}

log "waiting for kubelet to build a new sandbox under the same pod UID"
if ! wait_for "new sandbox" sandbox_recreated; then
  dump_state
  exit 1
fi
SANDBOX_AFTER=$(newest_sandbox)
IP_AFTER=$(pod_field '{.status.podIP}')
RESTARTS_AFTER=$(pod_field '{.status.containerStatuses[0].restartCount}')
log "recreated: sandbox ${SANDBOX_BEFORE:0:12} -> ${SANDBOX_AFTER:0:12}, ip $IP_BEFORE -> $IP_AFTER, restarts $RESTARTS_BEFORE -> $RESTARTS_AFTER"

# --- verdict: checkpoints after recreation must be attached and moving -------

T_NEW=$(now_ms)
enough_rows() { (( $(rows_since "$T_NEW") >= VERDICT_ROWS + SETTLE_ROWS )); }
log "waiting for $((VERDICT_ROWS + SETTLE_ROWS)) checkpoints written after the new sandbox came up"
wait_for "post-recreation checkpoints" enough_rows

ROWS=$(ch "SELECT ts, event_kind, ifNull(attributes.network_attached::Nullable(Bool), false),
                  network_egress_public_bytes + network_egress_private_bytes
           FROM instance_checkpoints_v1
           WHERE workspace_id = '$WORKSPACE' AND pod_uid = '$POD_UID' AND ts >= $T_KILL
           ORDER BY ts")
printf '\nts_ms\tkind\tattached\tegress_bytes\n%s\n\n' "$ROWS"

LAST=$(printf '%s\n' "$ROWS" | tail -n "$VERDICT_ROWS")
UNATTACHED=$(printf '%s\n' "$LAST" | awk -F'\t' '$3 != "true"' | wc -l | tr -d ' ')
FIRST_EGRESS=$(printf '%s\n' "$LAST" | head -n1 | cut -f4)
LAST_EGRESS=$(printf '%s\n' "$LAST" | tail -n1 | cut -f4)

# Informational. "netns replaced" only appears when the exit event was
# missed and Attach noticed the swap itself; the OOM path usually goes
# exit -> Detach -> "sandbox not ready" -> fresh attach.
log "heimdall log lines for this pod since the kill:"
kubectl -n "$HEIMDALL_NS" logs "$HEIMDALL_POD" --since="$(( ($(now_ms) - T_KILL) / 1000 + 5 ))s" 2>/dev/null \
  | grep -E "$POD_UID|netns replaced|network attach" | tail -n 20 || true
echo

if (( UNATTACHED > 0 )); then
  fail "$UNATTACHED of the last $VERDICT_ROWS checkpoints have network_attached=false after sandbox recreation"
fi
ok "last $VERDICT_ROWS checkpoints after recreation have network_attached=true"

if (( LAST_EGRESS <= FIRST_EGRESS )); then
  fail "egress counter did not grow after recreation ($FIRST_EGRESS -> $LAST_EGRESS): attached to a dead netns"
fi
ok "egress counter grows after recreation ($FIRST_EGRESS -> $LAST_EGRESS bytes): new netns is measured"

ok "heimdall keeps metering across gVisor sandbox recreation"
