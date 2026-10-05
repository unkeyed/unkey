#!/usr/bin/env bash
# Start only the Next dashboard for the local sign-in document.
# Does not start Docker, MySQL, or the API.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

"$SKILL_DIR/scripts/doctor.sh" preflight-auth

cd "$ROOT"
python3 -c 'import os,sys; os.setsid(); os.execvp("mise", ["mise", "exec", "--", "pnpm", "--dir=web/apps/dashboard", "dev"])' \
  >"$LOG_FILE" 2>&1 </dev/null &
printf '%s\n' "$!" >"$PID_FILE"
printf 'auth-landing\n' >"$MODE_FILE"

pid="$(recorded_pid)"
deadline=$((SECONDS + 180))
while (( SECONDS < deadline )); do
  if ! pid_alive "$pid"; then
    printf 'verdict: blocked\nAuth-landing process exited. Log: %s\n' "$LOG_FILE" >&2
    tail -n 80 "$LOG_FILE" >&2 || true
    exit 4
  fi
  if "$SKILL_DIR/scripts/doctor.sh" ready >/dev/null 2>&1; then
    "$SKILL_DIR/scripts/doctor.sh" ready
    exit 0
  fi
  sleep 2
done

printf 'verdict: blocked\nTimed out waiting for %s/auth/sign-in. Log: %s\n' "$BASE_URL" "$LOG_FILE" >&2
tail -n 80 "$LOG_FILE" >&2 || true
exit 4
