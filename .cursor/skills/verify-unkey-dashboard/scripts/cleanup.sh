#!/usr/bin/env bash
# Stop the process group this skill started. Does not delete evidence/.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

pid="$(recorded_pid)"
mode="$(recorded_mode)"

if pid_alive "$pid"; then
  kill -TERM -"$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    pid_alive "$pid" || break
    sleep 0.5
  done
  if pid_alive "$pid"; then
    kill -KILL -"$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null || true
  fi
  printf 'stopped pid %s mode %s\n' "$pid" "${mode:-unknown}"
else
  printf 'no live pid to stop (recorded: %s)\n' "${pid:-none}"
fi

if [[ "$mode" == "canonical" ]] && command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  docker compose -f "$COMPOSE_FILE" down
  printf 'docker compose down for %s\n' "$COMPOSE_FILE"
fi

rm -f "$PID_FILE" "$MODE_FILE"
printf 'evidence left in place under %s/evidence\n' "$SKILL_DIR"
