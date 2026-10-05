#!/usr/bin/env bash
# Start the documented dashboard stack. Records the session leader pid.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

"$SKILL_DIR/scripts/doctor.sh" preflight

cd "$ROOT"
python3 -c 'import os,sys; os.setsid(); os.execvp("mise", ["mise", "run", "dashboard"])' \
  >"$LOG_FILE" 2>&1 </dev/null &
printf '%s\n' "$!" >"$PID_FILE"
printf 'canonical\n' >"$MODE_FILE"

pid="$(recorded_pid)"
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if ! pid_alive "$pid"; then
    printf 'verdict: blocked\nCanonical launch exited early. Log: %s\n' "$LOG_FILE" >&2
    tail -n 80 "$LOG_FILE" >&2 || true
    exit 4
  fi
  sleep 1
done

printf 'verdict: launching\npid: %s\nlog: %s\nPoll: .cursor/skills/verify-unkey-dashboard/scripts/doctor.sh ready\nReadiness is HTTP 200 from %s/auth/sign-in with the text Local dashboard. mise run dashboard builds images and compiles Next before that.\n' "$pid" "$LOG_FILE" "$BASE_URL"
