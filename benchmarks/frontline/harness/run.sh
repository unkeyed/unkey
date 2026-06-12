#!/usr/bin/env bash
# Frontline load-test harness.
#
# Runs the proxy implementations as real, separate OS processes — load
# generator (hey), proxy under test, upstream, and MySQL each in their own
# process — so nothing shares a runtime, scheduler, or GC with the proxy.
#
# Scenarios (per implementation):
#   plain     GET /     over plain HTTP
#   tls       GET /     over TLS (self-signed localhost cert, static files)
#   tls16k    GET /16k  over TLS (16 KiB upstream response)
#
# Implementations:
#   stock     Go frontline built from origin/main (no local changes)
#   current   Go frontline built from the working tree
#
# Usage:
#   ./benchmarks/frontline/harness/run.sh                  # all impls
#   IMPLS="current" ./benchmarks/frontline/harness/run.sh
#   DURATION=20s WARMUP=5s CONC=64 ./benchmarks/frontline/harness/run.sh
#
# Requirements: docker, hey, go, openssl, curl.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
WORK="${WORK:-/tmp/frontline-bench}"
RESULTS="$WORK/results"
MYSQL_PORT="${MYSQL_PORT:-33306}"
UPSTREAM_PORT="${UPSTREAM_PORT:-38080}"
WARMUP="${WARMUP:-5s}"
DURATION="${DURATION:-20s}"
CONC="${CONC:-64}"
IMPLS="${IMPLS:-stock current}"
MYSQL_CONTAINER="unkey-bench-mysql"

# ---- performance knobs ----
# GOGC trades resident memory for fewer GC cycles; higher helps the
# larger-body (tls16k) path most. GOMEMLIMIT is left unset by default.
# GOAMD64 selects a microarch level and is a no-op unless GOARCH=amd64.
# PGO=auto captures a CPU profile from `current` under load and rebuilds it
# with profile-guided optimization; set PGO=off to skip.
GOGC="${GOGC:-200}"
GOMEMLIMIT="${GOMEMLIMIT:-}"
GOAMD64="${GOAMD64:-v3}"
PGO="${PGO:-auto}"
PGO_SECONDS="${PGO_SECONDS:-15}"
PGO_PROFILE="$ROOT/build/frontline/default.pgo"

# CPUPROFILE names the impl whose load runs are CPU-profiled (via its loopback
# pprof endpoint) into $RESULTS/<scenario>.cpu.prof, for `go tool pprof`.
# Profiling adds a few % overhead to *that* impl's measured rps, so set
# CPUPROFILE=off when you want a clean throughput comparison.
CPUPROFILE="${CPUPROFILE:-current}"

# Runtime env applied to every proxy launch.
RUNTIME_ENV=("GOGC=$GOGC")
[[ -n "$GOMEMLIMIT" ]] && RUNTIME_ENV+=("GOMEMLIMIT=$GOMEMLIMIT")

mkdir -p "$WORK" "$RESULTS"

log() { printf '\033[1;34m[harness]\033[0m %s\n' "$*" >&2; }

PIDS_TO_KILL=()
cleanup() {
  for pid in "${PIDS_TO_KILL[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT

# ---------------------------------------------------------------- MySQL ----
if ! docker inspect "$MYSQL_CONTAINER" >/dev/null 2>&1; then
  log "building MySQL image (schema auto-loads from pkg/mysql/schema)"
  docker build -q -f "$ROOT/dev/Dockerfile.mysql" -t unkey-bench-mysql "$ROOT" >/dev/null
  log "starting MySQL on port $MYSQL_PORT"
  docker run -d --name "$MYSQL_CONTAINER" \
    -p "$MYSQL_PORT:3306" \
    -e MYSQL_ROOT_PASSWORD=root \
    -e MYSQL_DATABASE=unkey \
    -e MYSQL_USER=unkey \
    -e MYSQL_PASSWORD=password \
    unkey-bench-mysql --max_connections=1000 --skip-log-bin >/dev/null
fi

# mysqladmin ping is not enough: the entrypoint brings up a temporary
# server while the schema init scripts run, then restarts. Wait until the
# unkey user can actually query a schema table.
log "waiting for MySQL schema to finish loading"
mysql_ready() {
  docker exec "$MYSQL_CONTAINER" mysql -uunkey -ppassword unkey \
    -e 'SELECT 1 FROM frontline_routes LIMIT 1' >/dev/null 2>&1
}
for _ in $(seq 1 90); do
  mysql_ready && break
  sleep 2
done
mysql_ready || { log "MySQL never became ready"; exit 1; }

log "seeding bench route"
docker exec -i "$MYSQL_CONTAINER" mysql -uunkey -ppassword unkey <"$ROOT/benchmarks/frontline/harness/seed.sql" 2>/dev/null

# ----------------------------------------------------------------- cert ----
if [[ ! -f "$WORK/cert.pem" ]]; then
  log "generating self-signed localhost certificate"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$WORK/key.pem" -out "$WORK/cert.pem" -days 7 \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" >/dev/null 2>&1
fi

# ------------------------------------------------------------- upstream ----
log "building + starting upstream on port $UPSTREAM_PORT"
(cd "$ROOT" && go build -o "$WORK/upstream" ./benchmarks/frontline/harness/upstream)
"$WORK/upstream" -port "$UPSTREAM_PORT" >"$WORK/upstream.log" 2>&1 &
PIDS_TO_KILL+=($!)
sleep 0.3
curl -sf "http://127.0.0.1:$UPSTREAM_PORT/" >/dev/null

# ------------------------------------------------------------- binaries ----
build_binary() {
  local impl="$1"
  case "$impl" in
  stock)
    if [[ ! -x "$WORK/frontline-stock" ]]; then
      log "building stock Go frontline from origin/main (git archive)"
      rm -rf "$WORK/stock-src" && mkdir -p "$WORK/stock-src"
      git -C "$ROOT" archive origin/main | tar -x -C "$WORK/stock-src"
      # stock is the origin/main baseline: the git archive does not contain
      # the working-tree default.pgo, so this build is never PGO-optimized.
      (cd "$WORK/stock-src" && GOAMD64="$GOAMD64" go build -o "$WORK/frontline-stock" ./build/frontline)
    fi
    ;;
  current)
    # go build defaults to -pgo=auto, so it transparently picks up
    # build/frontline/default.pgo when capture_pgo has written one.
    if [[ -f "$PGO_PROFILE" ]]; then
      log "building current Go frontline from working tree (PGO: $PGO_PROFILE)"
    else
      log "building current Go frontline from working tree (no PGO profile)"
    fi
    (cd "$ROOT" && GOAMD64="$GOAMD64" go build -o "$WORK/frontline-current" ./build/frontline)
    ;;
  esac
}

# -------------------------------------------------------------- configs ----
# Each (impl, mode) gets its own ports so runs can never collide.
port_base() {
  case "$1" in
  stock) echo 21000 ;;
  current) echo 22000 ;;
  esac
}

write_config() {
  local impl="$1" mode="$2" path="$3"
  local base http_port https_port prom_port pprof_port
  base="$(port_base "$impl")"
  case "$mode" in
  plain)
    http_port=$((base + 70)) https_port=$((base + 80)) prom_port=$((base + 90)) pprof_port=$((base + 92))
    ;;
  tls)
    http_port=$((base + 71)) https_port=$((base + 443)) prom_port=$((base + 91)) pprof_port=$((base + 93))
    ;;
  esac

  cat >"$path" <<EOF
platform = "local"
region = "bench"
http_port = $http_port
https_port = $https_port
prometheus_port = $prom_port
ctrl_addr = "localhost:1"
request_timeout = "1m"

[database]
primary = "unkey:password@tcp(127.0.0.1:$MYSQL_PORT)/unkey?parseTime=true"

[tls]
EOF
  if [[ "$mode" == "plain" ]]; then
    echo 'disabled = true' >>"$path"
  else
    echo "cert_file = \"$WORK/cert.pem\"" >>"$path"
    echo "key_file = \"$WORK/key.pem\"" >>"$path"
  fi
  # Loopback-only pprof server, used by capture_pgo to grab a CPU profile.
  {
    echo ""
    echo "[pprof]"
    echo "port = $pprof_port"
  } >>"$path"
  echo "$https_port"
}

# ------------------------------------------------------------ measuring ----
wait_ready() {
  local mode="$1" port="$2"
  for _ in $(seq 1 100); do
    if [[ "$mode" == "plain" ]]; then
      code="$(curl -s -o /dev/null -w '%{http_code}' -H 'Host: localhost' "http://127.0.0.1:$port/" || true)"
    else
      code="$(curl -sk -o /dev/null -w '%{http_code}' --connect-to "localhost:$port:127.0.0.1:$port" "https://localhost:$port/" || true)"
    fi
    [[ "$code" == "200" ]] && return 0
    sleep 0.2
  done
  return 1
}

# duration_seconds converts a hey-style duration ("20s", "1m") to an integer
# number of seconds for the pprof ?seconds= parameter.
duration_seconds() {
  case "$1" in
  *m) echo $(( ${1%m} * 60 )) ;;
  *s) echo "${1%s}" ;;
  *) echo "$1" ;;
  esac
}

# run_hey NAME URL [PPROF_PORT] — warmup (discarded), then measured run;
# extracts rps / p50 / p90 / p99 / non-200 count into $RESULTS/summary.tsv.
# When PPROF_PORT is given, a CPU profile is recorded over the measured run
# (not the warmup) into $RESULTS/NAME.cpu.prof.
run_hey() {
  local name="$1" url="$2" pprof_port="${3:-}"
  hey -z "$WARMUP" -c "$CONC" -host localhost "$url" >/dev/null 2>&1

  # Start the CPU profile here, after warmup, so it overlaps only the
  # measured load. Sized to DURATION; we wait for it before parsing.
  local cpu_pid=""
  if [[ -n "$pprof_port" ]]; then
    local secs
    secs="$(duration_seconds "$DURATION")"
    log "[$name] recording ${secs}s CPU profile -> $RESULTS/$name.cpu.prof"
    curl -sf "http://127.0.0.1:$pprof_port/_unkey/internal/pprof/profile?seconds=$secs" \
      -o "$RESULTS/$name.cpu.prof" 2>/dev/null &
    cpu_pid=$!
  fi

  hey -z "$DURATION" -c "$CONC" -host localhost "$url" >"$RESULTS/$name.txt" 2>&1

  if [[ -n "$cpu_pid" ]]; then
    wait "$cpu_pid" 2>/dev/null || true
  fi

  awk -v name="$name" '
    /Requests\/sec:/ { rps = $2 }
    /% in/ {
      gsub("%","",$1)
      if ($1 == "50") p50 = $3
      if ($1 == "90") p90 = $3
      if ($1 == "99") p99 = $3
    }
    /\[2[0-9][0-9]\]/ { ok += $2 }
    /\[[013-9][0-9][0-9]\]/ { bad += $2 }
    END {
      printf "%s\t%.0f\t%.1f\t%.1f\t%.1f\t%d\t%d\n",
        name, rps, p50*1000, p90*1000, p99*1000, ok, bad
    }
  ' "$RESULTS/$name.txt" >>"$RESULTS/summary.tsv"
}

bench_impl() {
  local impl="$1" bin="$WORK/frontline-$impl"

  # CPU-profile this impl's load runs? pprof ports mirror write_config:
  # plain = base+92, tls (shared by tls + tls16k) = base+93.
  local base prof_plain="" prof_tls=""
  base="$(port_base "$impl")"
  if [[ "$impl" == "$CPUPROFILE" ]]; then
    prof_plain=$((base + 92))
    prof_tls=$((base + 93))
  fi

  # plain instance
  local cfg="$WORK/$impl-plain.toml" port
  port="$(write_config "$impl" plain "$cfg")"
  log "[$impl] starting plain instance on :$port"
  env "${RUNTIME_ENV[@]}" UNKEY_LOG_LEVEL=error "$bin" --config "$cfg" >"$WORK/$impl-plain.log" 2>&1 &
  local pid=$!
  PIDS_TO_KILL+=($pid)
  wait_ready plain "$port" || { log "[$impl] plain instance never became ready (see $WORK/$impl-plain.log)"; return 1; }
  log "[$impl] plain: hey -z $DURATION -c $CONC"
  run_hey "$impl-plain" "http://127.0.0.1:$port/" "$prof_plain"
  echo -e "$impl-plain\t$(ps -o rss= -p $pid | tr -d ' ')" >>"$RESULTS/rss.tsv"
  kill $pid 2>/dev/null || true

  # tls instance
  cfg="$WORK/$impl-tls.toml"
  port="$(write_config "$impl" tls "$cfg")"
  log "[$impl] starting TLS instance on :$port"
  env "${RUNTIME_ENV[@]}" UNKEY_LOG_LEVEL=error "$bin" --config "$cfg" >"$WORK/$impl-tls.log" 2>&1 &
  pid=$!
  PIDS_TO_KILL+=($pid)
  wait_ready tls "$port" || { log "[$impl] tls instance never became ready (see $WORK/$impl-tls.log)"; return 1; }
  log "[$impl] tls: hey -z $DURATION -c $CONC"
  run_hey "$impl-tls" "https://127.0.0.1:$port/" "$prof_tls"
  log "[$impl] tls16k: hey -z $DURATION -c $CONC"
  run_hey "$impl-tls16k" "https://127.0.0.1:$port/16k" "$prof_tls"
  echo -e "$impl-tls\t$(ps -o rss= -p $pid | tr -d ' ')" >>"$RESULTS/rss.tsv"
  kill $pid 2>/dev/null || true
}

# ------------------------------------------------------------------ pgo ----
# Capture a CPU profile from `current` under representative load and write it
# to build/frontline/default.pgo. `go build` then auto-applies it (-pgo=auto)
# when (re)building `current`. Requires upstream + MySQL to already be up.
capture_pgo() {
  log "PGO: building unprofiled current binary for profiling"
  (cd "$ROOT" && GOAMD64="$GOAMD64" go build -pgo=off -o "$WORK/frontline-current-pgo" ./build/frontline)

  local cfg="$WORK/current-pgo.toml" port pprof_port
  port="$(write_config current plain "$cfg")"
  pprof_port=$(($(port_base current) + 92))

  env "${RUNTIME_ENV[@]}" UNKEY_LOG_LEVEL=error "$WORK/frontline-current-pgo" --config "$cfg" >"$WORK/current-pgo.log" 2>&1 &
  local pid=$!
  PIDS_TO_KILL+=("$pid")
  if ! wait_ready plain "$port"; then
    log "PGO: instance never became ready (see $WORK/current-pgo.log); skipping PGO"
    kill "$pid" 2>/dev/null || true
    return 0
  fi

  log "PGO: recording ${PGO_SECONDS}s CPU profile under load (c=$CONC)"
  hey -z "$((PGO_SECONDS + 3))s" -c "$CONC" -host localhost "http://127.0.0.1:$port/" >/dev/null 2>&1 &
  local load_pid=$!
  if curl -sf "http://127.0.0.1:$pprof_port/_unkey/internal/pprof/profile?seconds=$PGO_SECONDS" -o "$PGO_PROFILE"; then
    log "PGO: wrote $PGO_PROFILE ($(wc -c <"$PGO_PROFILE" | tr -d ' ') bytes)"
  else
    log "PGO: capture failed; current will build without PGO"
    rm -f "$PGO_PROFILE"
  fi
  wait "$load_pid" 2>/dev/null || true
  kill "$pid" 2>/dev/null || true
}

# ------------------------------------------------------------------ run ----
: >"$RESULTS/summary.tsv"
: >"$RESULTS/rss.tsv"

log "performance knobs: GOGC=$GOGC GOMEMLIMIT=${GOMEMLIMIT:-unset} GOAMD64=$GOAMD64 PGO=$PGO CPUPROFILE=${CPUPROFILE:-off}"

log "baseline: direct to upstream"
run_hey "baseline-direct" "http://127.0.0.1:$UPSTREAM_PORT/"

if [[ "$PGO" == "auto" && " $IMPLS " == *" current "* ]]; then
  capture_pgo
fi

for impl in $IMPLS; do
  build_binary "$impl"
  bench_impl "$impl"
done

# ---------------------------------------------------------------- report ----
echo
echo "scenario          rps        p50ms   p90ms   p99ms   2xx       non-2xx"
echo "----------------- ---------- ------- ------- ------- --------- -------"
awk -F'\t' '{ printf "%-17s %-10s %-7s %-7s %-7s %-9s %s\n", $1, $2, $3, $4, $5, $6, $7 }' "$RESULTS/summary.tsv"
echo
echo "RSS after load (KB):"
awk -F'\t' '{ printf "  %-17s %s\n", $1, $2 }' "$RESULTS/rss.tsv"
echo
if ls "$RESULTS"/*.cpu.prof >/dev/null 2>&1; then
  echo "CPU profiles (go tool pprof -http=: <file>):"
  for prof in "$RESULTS"/*.cpu.prof; do
    echo "  $prof"
  done
  echo
fi
echo "raw hey output: $RESULTS/"
