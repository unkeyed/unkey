#!/usr/bin/env bash
set -euo pipefail

command -v stripe >/dev/null 2>&1 || exit 0
output=$(mktemp)
cleanup() {
  kill "${stripe_pid:-}" "${timer_pid:-}" 2>/dev/null || true
  wait 2>/dev/null || true
  rm -f "$output"
}
trap cleanup EXIT
trap 'exit 0' INT TERM

stripe listen --print-secret --skip-update >"$output" 2>/dev/null </dev/null &
stripe_pid=$!
(
  sleep 5 &
  sleep_pid=$!
  trap 'kill "$sleep_pid" 2>/dev/null || true; wait "$sleep_pid" 2>/dev/null || true; exit 0' TERM
  wait "$sleep_pid"
  kill -KILL "$stripe_pid" 2>/dev/null || true
) &
timer_pid=$!

if wait "$stripe_pid" 2>/dev/null; then
  secret=$(<"$output")
  if [[ "$secret" =~ ^whsec_[a-zA-Z0-9]+$ ]]; then
    printf '%s' "$secret"
  fi
fi
stripe_pid=
