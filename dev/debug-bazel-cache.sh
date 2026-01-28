#!/usr/bin/env bash
# Debug bazel remote cache uploads/downloads per target.
#
# Usage:
#   ./dev/debug-bazel-cache.sh build //...
#   ./dev/debug-bazel-cache.sh test //pkg/cache:cache_test
#   ./dev/debug-bazel-cache.sh build //svc/api:api
#
set -euo pipefail

EXEC_LOG="/tmp/bazel_exec_$$.json"

cleanup() {
  rm -f "$EXEC_LOG"
}
trap cleanup EXIT

echo "Running: bazel $*"
bazel "$@" --execution_log_json_file="$EXEC_LOG"

echo ""
echo "=== Remote Cache Stats ==="
jq -rs '
  def sum_sizes: [.[].digest.sizeBytes | tonumber] | if length == 0 then 0 else add end;
  def to_mb: . / 1048576;
  group_by(.targetLabel) | 
  map({
    target: .[0].targetLabel,
    up: ([.[] | select(.cacheHit == false) | .actualOutputs[]] | sum_sizes | to_mb | floor),
    down: ([.[] | select(.cacheHit == true) | .actualOutputs[]] | sum_sizes | to_mb | floor)
  }) |
  map(select(.up > 1 or .down > 1)) |
  sort_by(-.up - .down) |
  .[] | 
  "\(.target)\t↑\(.up)MB\t↓\(.down)MB"
' "$EXEC_LOG" | column -t -s $'\t'

echo ""
jq -rs '
  def sum_sizes: [.[].digest.sizeBytes | tonumber] | if length == 0 then 0 else add end;
  def to_mb: . / 1048576;
  {
    uploaded: ([.[] | select(.cacheHit == false) | .actualOutputs[]] | sum_sizes | to_mb),
    downloaded: ([.[] | select(.cacheHit == true) | .actualOutputs[]] | sum_sizes | to_mb)
  } |
  "Total: ↑\(.uploaded | floor)MB ↓\(.downloaded | floor)MB"
' "$EXEC_LOG"
