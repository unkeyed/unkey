import { z } from "zod";
import type { Querier } from "../client";
import { DAY_MS, type Grid, type Rollup, SETTLE_MS, pickRollup } from "./grid";

export const RESOURCE_ROLLUPS: readonly Rollup[] = [
  { table: "instance_resources_per_minute_v1", bucketMs: 60 * 1000, ttlMs: 30 * DAY_MS },
  { table: "instance_resources_per_hour_v1", bucketMs: 60 * 60 * 1000, ttlMs: 90 * DAY_MS },
];

export const resourcePoint = z.object({
  t: z.number().int(),
  cpu: z.number().nullable(),
  memory: z.number().nullable(),
  disk: z.number().nullable(),
  ingress: z.number().nullable(),
  egress: z.number().nullable(),
  instances: z.number().int().nullable(),
});

/**
 * One bucket of an app or instance. cpu is millicores; ingress and egress are
 * public plus private bytes per second. For the app, memory, disk and
 * instances are the highest concurrent total in the bucket, so a rollout
 * counts both sides only while both run. For one instance, memory and disk
 * are its peak and instances is 1. A bucket where no container reported has
 * every field null: a gap, not a zero.
 */
export type ResourcePoint = z.infer<typeof resourcePoint>;

const instanceResourcePoint = resourcePoint.extend({ instance: z.string() });
export type InstanceResourcePoint = z.infer<typeof instanceResourcePoint>;

export type AppResourcesRequest = {
  workspaceId: string;
  environmentId: string;
  grid: Grid;
};

export type AppResourcesByInstanceRequest = AppResourcesRequest & { instances: string[] };

const params = z.object({
  workspaceId: z.string(),
  environmentId: z.string(),
  table: z.string(),
  gaugeTable: z.string(),
  sourceSec: z.number().int(),
  bucketSec: z.number().int(),
  readStartSec: z.number().int(),
  startSec: z.number().int(),
  endSec: z.number().int(),
  startMs: z.number().int(),
  endMs: z.number().int(),
  bucketMs: z.number().int(),
});

const instanceParams = params.extend({ instances: z.array(z.string()) });

// The deployment ids come from the day rollup, which every resource MV fills
// from the same checkpoint insert, so the set is exact and the resource_id
// sort key still prunes the minute table. app_id is not filtered: older rows
// carry an empty app_id, and the scope loader already checked that the
// environment belongs to the app.
const sourceRows = (filter: string, table = "{table: Identifier}") => `
  FROM ${table}
  WHERE workspace_id = {workspaceId: String}
    AND resource_type = 'deployment'
    AND environment_id = {environmentId: String}
    AND resource_id IN (
      SELECT DISTINCT resource_id
      FROM instance_resources_per_day_v1
      WHERE workspace_id = {workspaceId: String}
        AND resource_type = 'deployment'
        AND environment_id = {environmentId: String}
        AND time >= toStartOfDay(toDateTime({readStartSec: UInt32}))
        AND time <  toDateTime({endSec: UInt32})
    )${filter}`;

// max(max) - min(min) inside a bucket drops the step between the previous
// bucket's last sample and this bucket's first. The counter delta is measured
// from the previous bucket's max only when that bucket is the one directly
// before, so the delta never spans more than the bucket it is divided by. That
// is why one extra leading bucket is read and dropped.
const counterDelta = (mx: string, mn: string) =>
  `greatest(0, ${mx} - if(b - lagInFrame(b) OVER w = {bucketSec: UInt32}, lagInFrame(${mx}) OVER w, ${mn}))`;

const instanceBuckets = (filter: string) => `
  SELECT
    instance,
    t,
    sum(cpu_delta) AS cpu_delta,
    sum(ingress_delta) AS ingress_delta,
    sum(egress_delta) AS egress_delta,
    sum(covered_sec) AS covered_sec,
    max(memory) AS memory,
    max(disk) AS disk
  FROM (
    SELECT
      b,
      toInt64(b) * 1000 AS t,
      instance,
      toUnixTimestamp(last_time) - toUnixTimestamp(first_time) + {sourceSec: UInt32} AS covered_sec,
      ${counterDelta("cpu_mx", "cpu_mn")} AS cpu_delta,
      memory,
      disk,
      ${counterDelta("ingress_mx", "ingress_mn")} AS ingress_delta,
      ${counterDelta("egress_mx", "egress_mn")} AS egress_delta
    FROM (
      SELECT
        intDiv(toUnixTimestamp(time), {bucketSec: UInt32}) * {bucketSec: UInt32} AS b,
        container_uid,
        any(instance_id) AS instance,
        min(time) AS first_time,
        max(time) AS last_time,
        max(cpu_usage_usec_max) AS cpu_mx,
        min(cpu_usage_usec_min) AS cpu_mn,
        max(memory_bytes_max) AS memory,
        max(disk_used_bytes_max) AS disk,
        max(network_ingress_public_bytes_max) + max(network_ingress_private_bytes_max) AS ingress_mx,
        min(network_ingress_public_bytes_min) + min(network_ingress_private_bytes_min) AS ingress_mn,
        max(network_egress_public_bytes_max) + max(network_egress_private_bytes_max) AS egress_mx,
        min(network_egress_public_bytes_min) + min(network_egress_private_bytes_min) AS egress_mn
      ${sourceRows(filter)}
        AND time >= toDateTime({readStartSec: UInt32})
        AND time <  toDateTime({endSec: UInt32})
      GROUP BY b, container_uid
    )
    WINDOW w AS (PARTITION BY container_uid ORDER BY b ROWS BETWEEN 1 PRECEDING AND CURRENT ROW)
  )
  WHERE b >= {startSec: UInt32}
  GROUP BY instance, t`;

const FILL = `
  WITH FILL
    FROM {startMs: Int64}
    TO {endMs: Int64}
    STEP {bucketMs: Int64}`;

// Rates are a time-weighted average over the whole bucket, so sequential
// containers and rollout overlap add up correctly.
//
// Gauges are summed per minute, then the bucket keeps the highest sum, so a
// rollout counts both sides only for the minutes both ran. They always read the
// minute rollup: an hourly row cannot tell a pod that ran the whole hour from
// one that ran a minute of it.
const APP_QUERY = `
  SELECT
    t,
    toNullable(sum(cpu_delta) / {bucketSec: UInt32} / 1000) AS cpu,
    any(gauges.memory) AS memory,
    any(gauges.disk) AS disk,
    toNullable(sum(ingress_delta) / {bucketSec: UInt32}) AS ingress,
    toNullable(sum(egress_delta) / {bucketSec: UInt32}) AS egress,
    any(gauges.instances) AS instances
  FROM (${instanceBuckets("")}
  ) AS rates
  LEFT JOIN (
    SELECT
      toInt64(b) * 1000 AS t,
      toNullable(max(memory)) AS memory,
      toNullable(max(disk)) AS disk,
      toNullable(toUInt32(max(instances))) AS instances
    FROM (
      SELECT
        b,
        time,
        sum(memory) AS memory,
        sum(disk) AS disk,
        count() AS instances
      FROM (
        SELECT
          intDiv(toUnixTimestamp(time), {bucketSec: UInt32}) * {bucketSec: UInt32} AS b,
          time,
          max(memory_bytes_max) AS memory,
          max(disk_used_bytes_max) AS disk
        ${sourceRows("", "{gaugeTable: Identifier}")}
          AND time >= toDateTime({startSec: UInt32})
          AND time <  toDateTime({endSec: UInt32})
        GROUP BY b, time, instance_id
      )
      GROUP BY b, time
    )
    GROUP BY b
  ) AS gauges USING t
  GROUP BY t
  ORDER BY t ASC${FILL}`;

// One instance is averaged over the seconds its containers ran, so a pod that
// started, stopped or restarted inside the bucket does not read as idle for the
// rest. On the hourly rollup (3d and up) a container's covered span is always
// the full hour, so that correction only applies to minute-rollup ranges.
// WITH FILL over no rows at all emits one series with an empty instance, which
// the outer filter drops.
const INSTANCE_QUERY = `
  SELECT *
  FROM (
    SELECT
      instance,
      t,
      toNullable(sum(cpu_delta) / least(sum(covered_sec), {bucketSec: UInt32}) / 1000) AS cpu,
      toNullable(toFloat64(max(memory))) AS memory,
      toNullable(toFloat64(max(disk))) AS disk,
      toNullable(sum(ingress_delta) / least(sum(covered_sec), {bucketSec: UInt32})) AS ingress,
      toNullable(sum(egress_delta) / least(sum(covered_sec), {bucketSec: UInt32})) AS egress,
      toNullable(toUInt32(count())) AS instances
    FROM (${instanceBuckets(`
      AND instance_id IN {instances: Array(String)}`)}
    )
    GROUP BY instance, t
    ORDER BY instance, t ASC${FILL}
  )
  WHERE instance != ''
  ORDER BY instance, t ASC`;

function toParams(req: AppResourcesRequest, nowMs: number) {
  const { grid } = req;
  const bucketSec = grid.bucketMs / 1000;
  const startSec = grid.startMs / 1000;
  const rollup = pickRollup(RESOURCE_ROLLUPS, grid, nowMs);
  // A grid ends up to one bucket plus SETTLE_MS before now, so this is the
  // oldest start the minute rollup still covers except for part of the first
  // bucket, which only lowers that bucket's peak. Older windows read the hourly
  // rows, where a rollout hour counts both sides.
  const [minute] = RESOURCE_ROLLUPS;
  const gaugeRollup =
    nowMs - grid.startMs <= minute.ttlMs + grid.bucketMs + SETTLE_MS ? minute : rollup;
  return {
    workspaceId: req.workspaceId,
    environmentId: req.environmentId,
    table: rollup.table,
    gaugeTable: gaugeRollup.table,
    sourceSec: rollup.bucketMs / 1000,
    bucketSec,
    readStartSec: startSec - bucketSec,
    startSec,
    endSec: grid.endMs / 1000,
    startMs: grid.startMs,
    endMs: grid.endMs,
    bucketMs: grid.bucketMs,
  };
}

export function getAppResources(ch: Querier) {
  const query = ch.query({ query: APP_QUERY, params, schema: resourcePoint });
  return (req: AppResourcesRequest, nowMs = Date.now()) => query(toParams(req, nowMs));
}

/**
 * Reads only the named pods. An environment that deploys often has seen far
 * more pods than it runs, and each one would fill to a full grid.
 */
export function getAppResourcesByInstance(ch: Querier) {
  const query = ch.query({
    query: INSTANCE_QUERY,
    params: instanceParams,
    schema: instanceResourcePoint,
  });
  return (req: AppResourcesByInstanceRequest, nowMs = Date.now()) =>
    query({ ...toParams(req, nowMs), instances: req.instances });
}
