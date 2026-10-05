#!/usr/bin/env bash
# Read-only check. Does not start, stop, or modify processes or data.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  printf 'usage: doctor.sh preflight | preflight-auth | ready\n' >&2
  exit 64
}

[[ $# -eq 1 ]] || usage
mode="$1"

refuse_if_shared() {
  if our_instance_owns_listeners; then
    die refuse-shared "Dashboard already running as pid $(recorded_pid) mode $(recorded_mode). Do not start a second instance. Run: .cursor/skills/verify-unkey-dashboard/scripts/doctor.sh ready" 2
  fi
  local ports containers
  ports="$(foreign_ports || true)"
  containers="$(foreign_containers || true)"
  if [[ -n "$ports" || -n "$containers" ]]; then
    die refuse-shared "Refusing to drive a shared instance. Ports in use: ${ports:-none}. Containers present: ${containers:-none}. These names and ports are fixed by web/apps/dashboard/dev/docker-compose.yaml and next dev on port ${PORT}." 2
  fi
}

sign_in_body() {
  curl -fsS --max-time 10 "${BASE_URL}/auth/sign-in" 2>/dev/null || true
}

case "$mode" in
  preflight)
    command -v mise >/dev/null 2>&1 || die blocked "mise is not on PATH. Install it with ./dev/install-mise from the repo root. Canonical launch is mise run dashboard." 4
    command -v docker >/dev/null 2>&1 || die blocked "docker is not on PATH. mise run dashboard starts web/apps/dashboard/dev/docker-compose.yaml and cannot boot without it." 4
    docker info >/dev/null 2>&1 || die blocked "docker info failed. The daemon is not usable, so the dashboard stack cannot start." 4
    require_local_auth
    [[ -n "$(env_value DATABASE_PRIMARY)" ]] || die blocked "DATABASE_PRIMARY is empty. Set it to the local compose URL in web/apps/dashboard/dev/.env.example before mise run dashboard. Do not use a hosted database URL." 4
    refuse_if_shared
    printf 'verdict: safe-to-launch\nAUTH_PROVIDER=local\nport %s is free\ndocker is usable\n' "$PORT"
    ;;
  preflight-auth)
    command -v mise >/dev/null 2>&1 || die blocked "mise is not on PATH. Install it with ./dev/install-mise. Auth-landing launch uses mise exec -- pnpm." 4
    require_local_auth
    refuse_if_shared
    printf 'verdict: safe-to-launch-auth\nAUTH_PROVIDER=local\nport %s is free\nThis check does not start MySQL. Only features/auth-gate.md may be driven, and only the sign-in document.\n' "$PORT"
    ;;
  ready)
    pid="$(recorded_pid)"
    run_mode="$(recorded_mode)"
    if ! pid_alive "$pid"; then
      if port_open "$PORT"; then
        die refuse-shared "Port ${PORT} is open but recorded pid '${pid:-none}' is not alive. Refusing to attach to an unknown dashboard." 2
      fi
      die not-running "No dashboard process from this skill is alive." 3
    fi
    if ! port_open "$PORT"; then
      die not-running "pid ${pid} is alive but port ${PORT} is closed. Read ${LOG_FILE}." 3
    fi
    body="$(sign_in_body)"
    if [[ "$body" != *"Local dashboard"* || "$body" != *"Continue to dashboard"* ]]; then
      die blocked "GET ${BASE_URL}/auth/sign-in did not return the local landing (heading Local dashboard, link Continue to dashboard). The process is not the local-auth dashboard." 4
    fi
    if [[ "$run_mode" == "canonical" ]] && port_open 3306 && port_open 7070 && [[ -n "$(env_value DATABASE_PRIMARY)" ]]; then
      printf 'verdict: drivable\npid: %s\nmode: canonical\nready: %s/auth/sign-in\n' "$pid" "$BASE_URL"
    else
      printf 'verdict: auth-landing-only\npid: %s\nmode: %s\nready: %s/auth/sign-in contains Local dashboard\nMySQL or the API is not part of this instance. Drive features/auth-gate.md only. Do not activate Continue to dashboard.\n' "$pid" "${run_mode:-unknown}" "$BASE_URL"
    fi
    ;;
  *)
    usage
    ;;
esac
