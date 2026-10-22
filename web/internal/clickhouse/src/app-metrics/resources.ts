import { z } from "zod";
import type { Querier } from "../client";
import { DAY_MS, type Grid, type Rollup, pickRollup } from "./grid";

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
 * One bucket summed over every container that reported in it, so a rollout
 * overlap counts both sides. cpu is millicores; memory is peak working-set
 * bytes; disk is peak used bytes; ingress and egress are public plus private
 * bytes per second. instances counts the reporting instances. A bucket where
 * no container reported has every field null: a gap, not a zero.
 */
export type ResourcePoint = z.infer<typeof resourcePoint>;

const instanceResourcePoint = resourcePoint.extend({ instance: z.string() });
export type InstanceResourcePoint = z.infer<typeof instanceResourcePoint>;

export type AppResourcesRequest = {
  workspaceId: string;
  environmentId: string;
  grid: Grid;
};

const params = z.object({
  workspaceId: z.string(),
  environmentId: z.string(),
  table: z.string(),
  sourceSec: z.number().int(),
  bucketSec: z.number().int(),
  readStartSec: z.number().int(),
  startSec: z.number().int(),
  endSec: z.number().int(),
  startMs: z.number().int(),
  endMs: z.number().int(),
  bucketMs: z.number().int(),
});

// max(max) - min(min) inside a bucket drops the step between the previous
// bucket's last sample and this bucket's first. The counter delta is measured
// from the previous bucket's max only when that bucket is the one directly
// before, so the delta never spans more than the bucket it is divided by. That
// is why one extra leading bucket is read and dropped.
const counterDelta = (mx: string, mn: string) =>
  `greatest(0, ${mx} - if(b - lagInFrame(b) OVER w = {bucketSec: UInt32}, lagInFrame(${mx}) OVER w, ${mn}))`;

type Level = "app" | "instance";

// Rate columns. The app is a time-weighted average over the whole bucket, so
// sequential containers and rollout overlap add up correctly. One instance is
// averaged over the seconds its containers ran, so a pod that started,
// stopped or restarted inside the bucket does not read as idle for the rest.
// On the hourly rollup (3d and up) a container's covered span is always the
// full hour, so that correction only applies to minute-rollup ranges.
// Peak columns (memory, disk) take each instance's max across its
// containers; the app sums those per-instance peaks.
const OUTER: Record<Level, { key: string; rate: (delta: string) => string }> = {
  app: {
    key: "",
    rate: (delta) => `sum(${delta}) / {bucketSec: UInt32}`,
  },
  instance: {
    key: "instance, ",
    rate: (delta) => `sum(${delta}) / least(sum(covered_sec), {bucketSec: UInt32})`,
  },
};

// The deployment ids come from the day rollup, which every resource MV fills
// from the same checkpoint insert, so the set is exact and the resource_id
// sort key still prunes the minute table. app_id is not filtered: older rows
// carry an empty app_id, and the scope loader already checked that the
// environment belongs to the app.
function buildQuery(by: Level): string {
  const { key, rate } = OUTER[by];
  return `
    SELECT
      ${key}t,
      toNullable(${rate("cpu_delta")} / 1000) AS cpu,
      toNullable(toFloat64(sum(memory))) AS memory,
      toNullable(toFloat64(sum(disk))) AS disk,
      toNullable(${rate("ingress_delta")}) AS ingress,
      toNullable(${rate("egress_delta")}) AS egress,
      toNullable(toUInt32(count())) AS instances
    FROM (
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
          FROM {table: Identifier}
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
            )
            AND time >= toDateTime({readStartSec: UInt32})
            AND time <  toDateTime({endSec: UInt32})
          GROUP BY b, container_uid
        )
        WINDOW w AS (PARTITION BY container_uid ORDER BY b ROWS BETWEEN 1 PRECEDING AND CURRENT ROW)
      )
      WHERE b >= {startSec: UInt32}
      GROUP BY instance, t
    )
    GROUP BY ${key}t
    ORDER BY ${key}t ASC
    WITH FILL
      FROM {startMs: Int64}
      TO {endMs: Int64}
      STEP {bucketMs: Int64}`;
}

function toParams(req: AppResourcesRequest, nowMs: number) {
  const { grid } = req;
  const bucketSec = grid.bucketMs / 1000;
  const startSec = grid.startMs / 1000;
  const rollup = pickRollup(RESOURCE_ROLLUPS, grid, nowMs);
  return {
    workspaceId: req.workspaceId,
    environmentId: req.environmentId,
    table: rollup.table,
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
  const query = ch.query({ query: buildQuery("app"), params, schema: resourcePoint });
  return (req: AppResourcesRequest, nowMs = Date.now()) => query(toParams(req, nowMs));
}

export function getAppResourcesByInstance(ch: Querier) {
  const query = ch.query({
    query: buildQuery("instance"),
    params,
    schema: instanceResourcePoint,
  });
  return (req: AppResourcesRequest, nowMs = Date.now()) => query(toParams(req, nowMs));
}
