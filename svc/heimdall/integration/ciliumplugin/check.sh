#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
export KUBECONFIG="$root/.cache/cilium-integration/kubeconfig"
test "$(mise exec -- kubectl config current-context)" = heimdall-cilium
endpoint=$(mise exec -- kubectl -n heimdall-plugin-test get ciliumendpoint client -o jsonpath='{.status.id}')
allowed=$(mise exec -- kubectl -n heimdall-plugin-test get pod allowed -o jsonpath='{.status.podIP}')
denied=$(mise exec -- kubectl -n heimdall-plugin-test get pod denied -o jsonpath='{.status.podIP}')

inspect() {
  mise exec -- docker exec heimdall-cilium /opt/heimdall-plugin/plugin \
    -inspect "/sys/fs/bpf/cilium/endpoints/$endpoint"
}

before=$(inspect)
test "$(mise exec -- kubectl -n heimdall-plugin-test exec client -- wget -T 3 -qO- "http://$allowed:8080/")" = allowed
test "$(mise exec -- kubectl -n heimdall-plugin-test exec allowed -- wget -T 3 -qO- "http://$denied:8080/")" = denied
if mise exec -- kubectl -n heimdall-plugin-test exec client -- wget -T 3 -qO- "http://$denied:8080/"; then
  echo "FAIL: denied destination was reachable" >&2
  exit 1
fi
after=$(inspect)
mise exec -- jq -en --argjson before "$before" --argjson after "$after" '
  ($after.Counters.Before - $before.Counters.Before) as $pre |
  ($after.Counters.After - $before.Counters.After) as $post |
  ($after.Counters.Drop - $before.Counters.Drop) as $drop |
  ($after.Counters.Redirect - $before.Counters.Redirect) as $redirect |
  if $after.MapID == $before.MapID and $after.TargetID == $before.TargetID
    and $pre > 0 and $post == $pre and $drop > 0 and $redirect > 0
    and $post == $drop + $redirect
  then {result: "PASS", pre: $pre, post: $post, drop: $drop, redirect: $redirect}
  else error("unexpected verdicts or attachment changed during check") end'
