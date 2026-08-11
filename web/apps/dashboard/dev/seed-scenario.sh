#!/usr/bin/env bash
#
# seed-scenario.sh — put the local workspace ws_local into a named billing /
# usage / limits state, so the three settings pages can be reviewed state by
# state. Touches ws_local and its children only. Every run resets the rows it
# owns before writing, so switching or repeating a scenario never leaves a
# hybrid state.
#
#   no-plan         no Compute plan; Free API ceilings, light API usage
#   starter-over    Starter plan; compute allocation over all three workspace ceilings
#   pro-healthy     Pro plan; allocation ~40% of ceilings, ~$8 compute usage
#   business-high   Business plan; ~$200 compute usage, far past the $50 credit
#   suspended       Pro plan paused by the spend cap: $200 budget, stop on, spend_suspended=1
#   budget-no-stop  Pro plan, $150 budget, alerts only (spend_budget_stop=0)
#   budget-stop     Pro plan, $150 budget that stops workloads at 100%
#   api-over-quota  no Compute plan; API billable operations 225k against the 150k ceiling
#   both-over       Starter plan; compute allocation and API operations both breached
#   zero-usage      Pro plan, nothing deployed, no usage recorded anywhere
#   under-credit    Business plan, ~$45 usage, just under the $50 included credit
#   unattributed    Pro plan; half the compute usage carries blank project/app/env ids
#
# Usage:
#   ./seed-scenario.sh <name>
#   ./seed-scenario.sh list
#   ./seed-scenario.sh --watch    # apply whatever /tmp/unkey-seed-request asks for
#
# Not shippable as a scenario, because the dashboard reads these from Stripe and
# not from MySQL: a paid API tier (Upgrade -> Change), a scheduled cancellation,
# a past-due invoice, and any prorated fee or granted credit.

set -euo pipefail

# The marker names the live scenario; the dashboard banner reads it. The request
# file is how the dashboard asks for a different one: a web request must never
# run a shell script, so the page writes a name and --watch applies it.
MARKER=/tmp/unkey-seed-scenario
REQUEST=/tmp/unkey-seed-request
WS=ws_local
BASE_URL=https://limits-page.unkey.localhost/local/settings

SCENARIOS=(
  no-plan starter-over pro-healthy business-high suspended
  budget-no-stop budget-stop api-over-quota both-over
  zero-usage under-credit unattributed
)

# Fixed ids, so the reset is a delete by id and never touches seeded rows.
DEP_IDS=(d_scn1 d_scn2 d_scn3)
# A deployment the control plane rejected for want of CPU. Only the shapes that
# actually breach a ceiling get it, so the deploy-failure banner never appears in
# a workspace that has room.
FAILED_DEP_ID=d_cpulimit1
REGION=rgn_acmeuse1
# project:app:environment triples that already exist in the seed, so the Usage
# tree shows real names.
ANCHORS=(
  proj_gzkTZefKZ:app_RhtktLljk:env_wgGY1JEBU
  proj_DHuNhGvTh:app_aaOKmepeM:env_mD1KNWPK3
  proj_0Vmv1Cm8u:app_gzZApgPQB:env_tG5oGXMSI
)

# Meter rates from web/apps/dashboard/lib/billing/deployPricing.ts, in cents.
RATE_CPU_SECOND=0.0006944
RATE_MEMORY_GIB_SECOND=0.0003472
RATE_EGRESS_GIB=5.0
RATE_DISK_GIB_SECOND=0.000006
RATE_ACTIVE_KEY=0.2
# Share of the meter spend each meter carries. Disk is priced so low that a
# larger share would need an absurd volume to move the bill.
MIX_CPU=0.465
MIX_MEMORY=0.465
MIX_DISK=0.005
MIX_EGRESS=0.065

mysql_do() {
  docker exec -i mysql mysql -uunkey -ppassword -D unkey --batch --raw 2>/dev/null
}

ch_do() {
  docker exec -i clickhouse clickhouse-client --user=default --password=password --multiquery
}

die() {
  echo "seed-scenario: $*" >&2
  exit 1
}

usage() {
  cat <<EOF
usage: $(basename "$0") <scenario>|list|--watch

scenarios:
$(printf '  %s\n' "${SCENARIOS[@]}")

--watch polls $REQUEST once a second and applies the scenario named in it.
EOF
}

known_scenario() {
  local candidate
  for candidate in "${SCENARIOS[@]}"; do
    [[ $candidate == "$1" ]] && return 0
  done
  return 1
}

list_scenarios() {
  local active=""
  [[ -f $MARKER ]] && active=$(tr -d '[:space:]' <"$MARKER")
  for name in "${SCENARIOS[@]}"; do
    if [[ $name == "$active" ]]; then
      printf '* %s\n' "$name"
    else
      printf '  %s\n' "$name"
    fi
  done
}

# --- month window -----------------------------------------------------------
# Both usage endpoints aggregate over the current UTC calendar month, and the
# hourly rollup is keyed on whole hours, so usage is written as whole hours
# ending at the last hour boundary that has already passed.
month_window() {
  YEAR=$(date -u +%Y)
  MONTH=$(date -u +%-m)
  MONTH_START=$(TZ=UTC date -j -f "%Y-%m-%d %H:%M:%S" "$(date -u +%Y-%m)-01 00:00:00" +%s)
  NOW=$(date -u +%s)
  WINDOW_END=$((NOW / 3600 * 3600))
  HOURS=$(((WINDOW_END - MONTH_START) / 3600))
  # Capped so a run late in the month keeps the per-hour figures realistic.
  ((HOURS > 336)) && HOURS=336
  # First hour of a month: one hour of data whose closing checkpoint is still in
  # the future, so the workspace total reads one sample pair short of the rollup.
  ((HOURS < 1)) && HOURS=1
  WINDOW_START=$((WINDOW_END - HOURS * 3600))
}

# --- reset ------------------------------------------------------------------
reset_state() {
  mysql_do <<SQL
DELETE FROM deployment_steps WHERE workspace_id = '$WS' AND deployment_id = '$FAILED_DEP_ID';
DELETE FROM deployments WHERE workspace_id = '$WS' AND id IN ('$FAILED_DEP_ID', 'd_cpuhog1');
DELETE FROM deployment_topology WHERE workspace_id = '$WS' AND deployment_id IN ('${DEP_IDS[0]}','${DEP_IDS[1]}','${DEP_IDS[2]}', 'd_cpuhog1');
DELETE FROM instances WHERE workspace_id = '$WS' AND deployment_id IN ('${DEP_IDS[0]}','${DEP_IDS[1]}','${DEP_IDS[2]}');
DELETE FROM cilium_network_policies WHERE workspace_id = '$WS' AND deployment_id IN ('${DEP_IDS[0]}','${DEP_IDS[1]}','${DEP_IDS[2]}');
DELETE FROM frontline_routes WHERE deployment_id IN ('${DEP_IDS[0]}','${DEP_IDS[1]}','${DEP_IDS[2]}');
DELETE FROM deployments WHERE workspace_id = '$WS' AND id IN ('${DEP_IDS[0]}','${DEP_IDS[1]}','${DEP_IDS[2]}');
-- A non-empty subscriptions blob routes billing to the legacy page, where none
-- of these states render.
UPDATE workspaces SET subscriptions = NULL WHERE id = '$WS';
SQL

  ch_do <<SQL
ALTER TABLE default.instance_usage_per_hour_v1 DELETE
  WHERE workspace_id = '$WS' AND time >= toDateTime($MONTH_START)
  SETTINGS mutations_sync = 2;
ALTER TABLE default.instance_checkpoints_v1 DELETE
  WHERE workspace_id = '$WS' AND ts >= ${MONTH_START}000
  SETTINGS mutations_sync = 2;
ALTER TABLE default.billable_verifications_per_month_v2 DELETE
  WHERE workspace_id = '$WS' AND year = $YEAR AND month = $MONTH
  SETTINGS mutations_sync = 2;
ALTER TABLE default.billable_ratelimits_per_month_v2 DELETE
  WHERE workspace_id = '$WS' AND year = $YEAR AND month = $MONTH
  SETTINGS mutations_sync = 2;
ALTER TABLE default.key_verifications_per_month_v3 DELETE
  WHERE workspace_id = '$WS' AND time = makeDate($YEAR, $MONTH, 1) AND source = 'gateway'
  SETTINGS mutations_sync = 2;
SQL
}

# --- billing row ------------------------------------------------------------
# stripe_customer_id is left alone: the payment method is workspace setup, not
# scenario state, and clearing it would replace every card with "add a payment
# method".
apply_billing() {
  local plan=$1 budget=$2 stop=$3 suspended=$4
  local plan_sql budget_sql
  plan_sql=$([[ -z $plan ]] && echo NULL || echo "'$plan'")
  budget_sql=$([[ -z $budget ]] && echo NULL || echo "$budget")

  mysql_do <<SQL
INSERT INTO workspace_billing
  (workspace_id, tier, plan, plan_override, spend_budget_cents, spend_budget_stop, spend_suspended, created_at_m, updated_at_m)
VALUES
  ('$WS', 'Free', $plan_sql, NULL, $budget_sql, $stop, $suspended, UNIX_TIMESTAMP() * 1000, UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE
  tier = 'Free', plan = $plan_sql, plan_override = NULL,
  spend_budget_cents = $budget_sql, spend_budget_stop = $stop, spend_suspended = $suspended,
  updated_at_m = UNIX_TIMESTAMP() * 1000, deleted_at_m = NULL;
SQL
}

# --- limits row -------------------------------------------------------------
# Values from limitsByPlan in web/apps/dashboard/lib/limits.ts.
apply_limits() {
  local plan=$1 ops logs audit team cpu cpu_i mem mem_i sto sto_i builds domains replicas
  ops=150000
  cpu=10
  mem=20480
  sto=51200
  sto_i=10240
  builds=1
  case $plan in
    free) logs=7 audit=30 team=0 cpu_i=2 mem_i=4096 domains=0 replicas=0 ;;
    starter) logs=3 audit=7 team=0 cpu_i=2 mem_i=2048 domains=1 replicas=4 ;;
    pro) logs=7 audit=14 team=1 cpu_i=8 mem_i=8192 domains=1000000 replicas=8 ;;
    business) logs=14 audit=30 team=1 cpu_i=16 mem_i=32768 domains=1000000 replicas=16 ;;
    *) die "unknown limits plan: $plan" ;;
  esac

  mysql_do <<SQL
INSERT INTO limits
  (workspace_id, api_billable_operations_count_max_per_month, api_requests_count_max_per_minute,
   logs_retention_days_max, logs_audit_retention_days_max, team_enabled,
   cpu_cores_max, cpu_cores_max_per_instance, memory_mib_max, memory_mib_max_per_instance,
   storage_mib_max, storage_mib_max_per_instance, builds_concurrent_max,
   custom_domains_max, autoscaling_replicas_max)
VALUES
  ('$WS', $ops, NULL, $logs, $audit, $team, $cpu, $cpu_i, $mem, $mem_i, $sto, $sto_i, $builds, $domains, $replicas)
ON DUPLICATE KEY UPDATE
  api_billable_operations_count_max_per_month = $ops, api_requests_count_max_per_minute = NULL,
  logs_retention_days_max = $logs, logs_audit_retention_days_max = $audit, team_enabled = $team,
  cpu_cores_max = $cpu, cpu_cores_max_per_instance = $cpu_i,
  memory_mib_max = $mem, memory_mib_max_per_instance = $mem_i,
  storage_mib_max = $sto, storage_mib_max_per_instance = $sto_i,
  builds_concurrent_max = $builds, custom_domains_max = $domains,
  autoscaling_replicas_max = $replicas;
SQL
}

# --- deployments ------------------------------------------------------------
# The Limits page reads compute in use as
# SUM(cpu_millicores * autoscaling_replicas_max) over running topology rows, so
# the deployment plus its topology row is what moves the figure. The instance,
# network policy and route rows come with any ready deployment.
# The spend cap does not merely flag a workspace: deployteardown SUSPENDs every
# running deployment, so a suspended workspace has drained and holds no
# allocation. Without this, Limits reported the workspace at its CPU ceiling
# while the paused banner said the workloads were stopped — two states that
# cannot both be true. desired_status is what queryComputeAllocation counts, so
# it is the one field that has to follow the suspension.
apply_deployments() {
  local shape=$1 suspended=$2 cpu mem sto replicas count topology_status
  topology_status=running
  if [[ $suspended == 1 ]]; then
    topology_status=stopped
  fi
  case $shape in
    none) return ;;
    healthy) cpu=1000 mem=2048 sto=4096 replicas=2 count=2 ;;
    mid) cpu=1000 mem=2048 sto=4096 replicas=2 count=3 ;;
    over) cpu=2000 mem=2048 sto=10240 replicas=4 count=3 ;;
    *) die "unknown deployment shape: $shape" ;;
  esac

  local now_ms=$(($(date +%s) * 1000))
  local i anchor project app env id
  for ((i = 0; i < count; i++)); do
    anchor=${ANCHORS[$i]}
    project=${anchor%%:*}
    app=${anchor#*:}
    app=${app%%:*}
    env=${anchor##*:}
    id=${DEP_IDS[$i]}

    mysql_do <<SQL
INSERT INTO deployments
  (id, k8s_name, workspace_id, project_id, environment_id, app_id, image,
   git_commit_sha, git_branch, git_commit_message, git_commit_author_handle,
   sentinel_config, cpu_millicores, memory_mib, storage_mib, desired_state,
   encrypted_environment_variables, status, \`trigger\`, created_at, updated_at)
VALUES
  ('$id', 'scn-$((i + 1))', '$WS', '$project', '$env', '$app', 'ghcr.io/unkey/scenario:$((i + 1))',
   'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa$((i + 1))', 'main', 'seed scenario deployment', 'unkey',
   _binary '{}', $cpu, $mem, $sto, 'running',
   _binary '', 'ready', 'cli', $now_ms, $now_ms);

INSERT INTO deployment_topology
  (workspace_id, deployment_id, region_id, autoscaling_replicas_min, autoscaling_replicas_max,
   desired_status, created_at, updated_at)
VALUES ('$WS', '$id', '$REGION', 1, $replicas, '$topology_status', $now_ms, $now_ms);

INSERT INTO instances
  (id, deployment_id, workspace_id, project_id, app_id, region_id, k8s_name, address,
   cpu_millicores, memory_mib, storage_mib, status)
VALUES
  ('ins_scn$((i + 1))', '$id', '$WS', '$project', '$app', '$REGION', 'scn-$((i + 1))-0',
   '10.42.$((i + 1)).10:8080', $cpu, $mem, $sto, 'running');

INSERT INTO cilium_network_policies
  (id, workspace_id, project_id, app_id, environment_id, deployment_id, k8s_name, k8s_namespace,
   region_id, policy, created_at, updated_at)
VALUES
  ('cnp_scn$((i + 1))', '$WS', '$project', '$app', '$env', '$id', 'scn-$((i + 1))', 'i4fm7rke',
   '$REGION', JSON_OBJECT('kind', 'CiliumNetworkPolicy'), $now_ms, $now_ms);

INSERT INTO frontline_routes
  (id, project_id, app_id, deployment_id, environment_id, fully_qualified_domain_name, sticky, created_at, updated_at)
VALUES
  ('flr_scn$((i + 1))', '$project', '$app', '$id', '$env', 'scn-$((i + 1)).unkey.app', 'deployment', $now_ms, $now_ms);
SQL
  done

  if [[ $shape == over ]]; then
    apply_failed_deployment
  fi
}

# The deployment the ceiling rejected. It has no topology and no instances: it
# never ran, so it holds no allocation and only the banner on its detail page has
# anything to say about it. The error text is the control plane's own
# deployfail.MsgCPUQuotaExceeded, which is what the dashboard matches on.
apply_failed_deployment() {
  local now_ms=$(($(date +%s) * 1000))
  local anchor=${ANCHORS[2]}
  local project=${anchor%%:*}
  local app=${anchor#*:}
  app=${app%%:*}
  local env=${anchor##*:}
  local started=$((now_ms - 600000))

  mysql_do <<SQL
INSERT INTO deployments
  (id, k8s_name, workspace_id, project_id, environment_id, app_id, image,
   git_commit_sha, git_branch, git_commit_message, git_commit_author_handle,
   sentinel_config, cpu_millicores, memory_mib, storage_mib, desired_state,
   encrypted_environment_variables, status, \`trigger\`, created_at, updated_at)
VALUES
  ('$FAILED_DEP_ID', 'd-cpulimit1', '$WS', '$project', '$env', '$app', 'ghcr.io/unkey/scenario:fail',
   'a1b2c3d4e5f60718293a4b5c6d7e8f9012345678', 'main', 'feat(api): add batch verify endpoint', 'unkey',
   _binary '{}', 2000, 2048, 4096, 'running',
   _binary '', 'failed', 'github', $started, $started);

INSERT INTO deployment_steps
  (workspace_id, project_id, environment_id, deployment_id, app_id, step, started_at, ended_at, error)
VALUES
  ('$WS', '$project', '$env', '$FAILED_DEP_ID', '$app', 'queued',    $started,          $((started + 1200)),  NULL),
  ('$WS', '$project', '$env', '$FAILED_DEP_ID', '$app', 'starting',  $((started + 1200)), $((started + 3400)),  NULL),
  ('$WS', '$project', '$env', '$FAILED_DEP_ID', '$app', 'building',  $((started + 3400)), $((started + 41000)), NULL),
  ('$WS', '$project', '$env', '$FAILED_DEP_ID', '$app', 'deploying', $((started + 41000)), $((started + 43500)),
   'We are unable to deploy this application as you have exceeded your CPU quota.');
SQL
}

# --- API operations ---------------------------------------------------------
# Billable API operations come from ClickHouse, not MySQL: the Limits page reads
# billing.queryUsage, which sums default.billable_verifications_per_month_v2 and
# default.billable_ratelimits_per_month_v2 for the calendar month.
#
# The active-key rows written by apply_compute_usage land in
# key_verifications_per_month_v3, which billable_verifications_per_month_mv_v2
# reads, so each one also counts as a billable verification. Their count is
# deducted here so the total is the figure the scenario declares.
apply_api_usage() {
  local verifications=$1 ratelimits=$2 active_keys=$3
  ((verifications == 0 && ratelimits == 0)) && return

  local direct=$((verifications - active_keys))
  ((direct < 0)) && direct=0

  ch_do <<SQL
INSERT INTO default.billable_verifications_per_month_v2 (year, month, workspace_id, count)
VALUES ($YEAR, $MONTH, '$WS', $direct);
INSERT INTO default.billable_ratelimits_per_month_v2 (year, month, workspace_id, count)
VALUES ($YEAR, $MONTH, '$WS', $ratelimits);
SQL
}

# --- compute usage ----------------------------------------------------------
# Two tables, deliberately kept in agreement: the Usage page breakdown reads the
# hourly rollup instance_usage_per_hour_v1, while the workspace total and the
# billing page read the raw instance_checkpoints_v1. Both are written from the
# same per-hour figures, so the tree adds up to the total beside it.
#
# Each container gets one checkpoint a minute across the window. Sixty pairs an
# hour with counters that advance by a fixed step per minute make the rollup row
# for that hour exact rather than approximate.
apply_compute_usage() {
  local target_cents=$1 containers=$2 active_keys=$3 attribution=$4
  ((target_cents == 0 && active_keys == 0)) && return

  if ((active_keys > 0)); then
    ch_do <<SQL
INSERT INTO default.key_verifications_per_month_v3
  (time, workspace_id, key_space_id, identity_id, external_id, key_id, outcome, source, tags, count, spent_credits)
SELECT makeDate($YEAR, $MONTH, 1), '$WS', 'ks_local', '', '',
       concat('key_scn_', toString(number)), 'VALID', 'gateway', [], 1, 0
FROM numbers($active_keys);
SQL
  fi

  ((target_cents == 0)) && return

  # Meter spend is the target less what the active keys already cost.
  local profile
  profile=$(awk -v target="$target_cents" -v keys="$active_keys" -v hours="$HOURS" \
    -v containers="$containers" -v key_rate="$RATE_ACTIVE_KEY" \
    -v r_cpu="$RATE_CPU_SECOND" -v r_mem="$RATE_MEMORY_GIB_SECOND" \
    -v r_egress="$RATE_EGRESS_GIB" -v r_disk="$RATE_DISK_GIB_SECOND" \
    -v m_cpu="$MIX_CPU" -v m_mem="$MIX_MEMORY" -v m_disk="$MIX_DISK" -v m_egress="$MIX_EGRESS" \
    'BEGIN {
       gib = 1073741824
       meters = target - keys * key_rate
       if (meters < 0) meters = 0
       per = meters / (hours * containers)
       cpu_usec_min = int(m_cpu * per / r_cpu * 1000000 / 60)
       mem_bytes    = int(m_mem * per / (3600 * r_mem) * gib)
       disk_bytes   = int(m_disk * per / (3600 * r_disk) * gib)
       egress_min   = int(m_egress * per / r_egress * gib / 60)
       printf "%d %d %d %d %.6f %.9f %.9f %d %d",
         cpu_usec_min, mem_bytes, disk_bytes, egress_min,
         cpu_usec_min * 60 / 1000000.0, mem_bytes / gib, disk_bytes / gib,
         egress_min * 60, int(cpu_usec_min * 60 / 3600000.0)
     }')
  read -r CPU_USEC_MIN MEM_BYTES DISK_BYTES EGRESS_MIN \
    CPU_SECONDS_HOUR MEM_GIB_HOURS DISK_GIB_HOURS EGRESS_BYTES_HOUR CPU_MILLICORES <<<"$profile"

  local c anchor project app env res cnt
  for ((c = 0; c < containers; c++)); do
    anchor=${ANCHORS[$((c % ${#ANCHORS[@]}))]}
    project=${anchor%%:*}
    app=${anchor#*:}
    app=${app%%:*}
    env=${anchor##*:}
    res=${DEP_IDS[$((c % ${#DEP_IDS[@]}))]}
    cnt="cnt_scn$((c + 1))"

    # The Usage page labels a blank id "Unattributed", so one container carries
    # none.
    if [[ $attribution == mixed && $c -eq $((containers - 1)) ]]; then
      project=""
      app=""
      env=""
    fi

    ch_do <<SQL
INSERT INTO default.instance_usage_per_hour_v1
  (time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id,
   container_uid, instance_id, cpu_seconds, memory_gib_hours, disk_gib_hours,
   network_egress_public_bytes, network_egress_private_bytes,
   network_ingress_public_bytes, network_ingress_private_bytes, sample_pairs, computed_at)
SELECT toDateTime($WINDOW_START + number * 3600), '$WS', '$project', '$app', '$env',
       'deployment', '$res', '$cnt', 'ins_$cnt',
       $CPU_SECONDS_HOUR, $MEM_GIB_HOURS, $DISK_GIB_HOURS, $EGRESS_BYTES_HOUR, 0, 0, 0, 60, now()
FROM numbers($HOURS);

INSERT INTO default.instance_checkpoints_v1
  (node_id, workspace_id, project_id, app_id, environment_id, resource_type, resource_id,
   pod_uid, instance_id, container_uid, restart_count, ts, event_kind,
   cpu_usage_usec, memory_bytes, cpu_allocated_millicores, memory_allocated_bytes,
   disk_allocated_bytes, disk_used_bytes,
   network_egress_public_bytes, network_egress_private_bytes,
   network_ingress_public_bytes, network_ingress_private_bytes, region, platform, attributes)
SELECT 'node-local', '$WS', '$project', '$app', '$env', 'deployment', '$res',
       'pod_$cnt', 'ins_$cnt', '$cnt', 0,
       ($WINDOW_START + number * 60) * 1000, 'periodic',
       number * $CPU_USEC_MIN, $MEM_BYTES, $CPU_MILLICORES, $MEM_BYTES,
       $DISK_BYTES, toInt64($DISK_BYTES * 0.4),
       number * $EGRESS_MIN, 0, 0, 0, 'us-east-1', 'aws',
       '{"network_attached":true}'::JSON
FROM numbers($((HOURS * 60 + 1)));
SQL
  done
}

# --- scenarios --------------------------------------------------------------
apply_scenario() {
  local name=$1
  local plan budget stop suspended limits shape verifications ratelimits
  local compute_cents containers active_keys attribution

  plan="" budget="" stop=0 suspended=0
  verifications=62000 ratelimits=18500
  compute_cents=0 containers=2 active_keys=0 attribution=real

  case $name in
    no-plan)
      limits=free shape=none
      ;;
    starter-over)
      plan=starter limits=starter shape=over
      compute_cents=900 containers=3 active_keys=300
      ;;
    pro-healthy)
      plan=pro limits=pro shape=healthy
      compute_cents=800 containers=2 active_keys=600
      ;;
    business-high)
      plan=business limits=business shape=mid
      compute_cents=20000 containers=3 active_keys=2000
      ;;
    suspended)
      plan=pro limits=pro shape=mid
      budget=20000 stop=1 suspended=1
      compute_cents=21500 containers=3 active_keys=1500
      ;;
    budget-no-stop)
      plan=pro limits=pro shape=healthy
      budget=15000 stop=0
      compute_cents=4200 containers=2 active_keys=600
      ;;
    budget-stop)
      plan=pro limits=pro shape=healthy
      budget=15000 stop=1
      compute_cents=4200 containers=2 active_keys=600
      ;;
    api-over-quota)
      limits=free shape=none
      verifications=180000 ratelimits=45000
      ;;
    both-over)
      plan=starter limits=starter shape=over
      verifications=180000 ratelimits=45000
      compute_cents=900 containers=3 active_keys=300
      ;;
    zero-usage)
      plan=pro limits=pro shape=none
      verifications=0 ratelimits=0
      ;;
    under-credit)
      plan=business limits=business shape=mid
      compute_cents=4500 containers=3 active_keys=800
      ;;
    unattributed)
      plan=pro limits=pro shape=healthy
      compute_cents=1200 containers=3 active_keys=400 attribution=mixed
      ;;
    *) die "unknown scenario: $name" ;;
  esac

  apply_billing "$plan" "$budget" "$stop" "$suspended"
  apply_limits "$limits"
  apply_deployments "$shape" "$suspended"
  apply_api_usage "$verifications" "$ratelimits" "$active_keys"
  apply_compute_usage "$compute_cents" "$containers" "$active_keys" "$attribution"
}

# --- watch ------------------------------------------------------------------
# Each request is applied by re-invoking this script, so watch and one-shot runs
# share one code path and a failure cannot leave the loop half-way through a
# scenario. SEED_SCENARIO_QUIET hands the printing to the loop.
watch_requests() {
  echo "seed-scenario: watching $REQUEST (ctrl-c to stop)"
  local name
  while true; do
    if [[ -f $REQUEST ]]; then
      name=$(tr -d '[:space:]' <"$REQUEST" || true)
      rm -f "$REQUEST"

      if [[ -z $name ]]; then
        printf '%s  ignored: empty request\n' "$(date '+%H:%M:%S')" >&2
      elif ! known_scenario "$name"; then
        printf '%s  ignored: unknown scenario %s\n' "$(date '+%H:%M:%S')" "$name" >&2
      elif SEED_SCENARIO_QUIET=1 "$0" "$name"; then
        printf '%s  applied %s\n' "$(date '+%H:%M:%S')" "$name"
      else
        rm -f "$MARKER"
        printf '%s  FAILED %s; no scenario is active\n' "$(date '+%H:%M:%S')" "$name" >&2
      fi
    fi
    sleep 1
  done
}

# --- main -------------------------------------------------------------------
main() {
  if (($# != 1)); then
    usage
    exit 1
  fi

  case $1 in
    list) list_scenarios; exit 0 ;;
    --watch) watch_requests; exit 0 ;;
    -h | --help) usage; exit 0 ;;
  esac

  local name=$1
  known_scenario "$name" || die "unknown scenario: $name (try 'list')"

  # Dropped up front and only rewritten on success, so a half-applied state
  # never claims to be a scenario.
  rm -f "$MARKER"
  trap 'rm -f "$MARKER"; echo "seed-scenario: $name failed; no scenario is active" >&2' ERR

  month_window
  reset_state
  apply_scenario "$name"

  printf '%s\n' "$name" >"$MARKER"
  trap - ERR

  if [[ -z ${SEED_SCENARIO_QUIET:-} ]]; then
    echo "applied: $name"
    echo "$BASE_URL/billing"
    echo "$BASE_URL/usage"
    echo "$BASE_URL/limits"
  fi
}

main "$@"
