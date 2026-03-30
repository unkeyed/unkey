import { z } from "zod";
import type { Querier } from "./client";

const SNAPSHOTS_TABLE = "default.instance_resource_snapshots_v1";
const PER_MINUTE_TABLE = "default.instance_resources_per_minute_v1";

const RESOURCE_FILTER = `
  workspace_id = {workspaceId: String}
  AND resource_type = {resourceType: String}
  AND resource_id = {resourceId: String}`;

const baseParams = z.object({
  workspaceId: z.string(),
  resourceType: z.enum(["deployment", "sentinel"]),
  resourceId: z.string(),
});

const timeseriesPointSchema = z.object({
  x: z.number().int(),
  y: z.number(),
});

// ─────────────────────────────────────────────────────────────
// CPU Timeseries
// ─────────────────────────────────────────────────────────────

export function getResourceCpuTimeseries(ch: Querier) {
  return async (args: z.infer<typeof baseParams> & { windowHours: number }) => {
    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(time) * 1000 AS x,
          cpu_millicores_sum / greatest(sample_count, 1) AS y
        FROM ${PER_MINUTE_TABLE}
        WHERE ${RESOURCE_FILTER}
          AND time >= now() - INTERVAL {windowHours: UInt8} HOUR
        ORDER BY time ASC`,
      params: baseParams.extend({ windowHours: z.number() }),
      schema: timeseriesPointSchema,
    });

    return query(args);
  };
}

// ─────────────────────────────────────────────────────────────
// Memory Timeseries
// ─────────────────────────────────────────────────────────────

export function getResourceMemoryTimeseries(ch: Querier) {
  return async (args: z.infer<typeof baseParams> & { windowHours: number }) => {
    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(time) * 1000 AS x,
          memory_bytes_max AS y
        FROM ${PER_MINUTE_TABLE}
        WHERE ${RESOURCE_FILTER}
          AND time >= now() - INTERVAL {windowHours: UInt8} HOUR
        ORDER BY time ASC`,
      params: baseParams.extend({ windowHours: z.number() }),
      schema: timeseriesPointSchema,
    });

    return query(args);
  };
}

// ─────────────────────────────────────────────────────────────
// Active Instances (replica count) Timeseries
// ─────────────────────────────────────────────────────────────

export function getResourceInstanceCountTimeseries(ch: Querier) {
  return async (args: z.infer<typeof baseParams> & { windowHours: number }) => {
    const query = ch.query({
      query: `
        SELECT
          toUnixTimestamp(time) * 1000 AS x,
          uniq(instance_id) AS y
        FROM ${PER_MINUTE_TABLE}
        WHERE ${RESOURCE_FILTER}
          AND time >= now() - INTERVAL {windowHours: UInt8} HOUR
        GROUP BY time
        ORDER BY time ASC`,
      params: baseParams.extend({ windowHours: z.number() }),
      schema: timeseriesPointSchema,
    });

    return query(args);
  };
}

// ─────────────────────────────────────────────────────────────
// Current Resource Summary (latest snapshots)
// ─────────────────────────────────────────────────────────────

const resourceSummarySchema = z.object({
  active_instances: z.number().int(),
  avg_cpu_millicores: z.number(),
  max_memory_bytes: z.number().int(),
  total_egress_bytes: z.number().int(),
  avg_cpu_limit_millicores: z.number(),
  avg_memory_limit_bytes: z.number(),
});

export function getResourceSummary(ch: Querier) {
  return async (args: z.infer<typeof baseParams>) => {
    const query = ch.query({
      query: `
        SELECT
          uniq(instance_id) AS active_instances,
          avg(cpu_millicores) AS avg_cpu_millicores,
          max(memory_bytes) AS max_memory_bytes,
          sum(network_egress_bytes) AS total_egress_bytes,
          avg(cpu_limit_millicores) AS avg_cpu_limit_millicores,
          avg(memory_limit_bytes) AS avg_memory_limit_bytes
        FROM ${SNAPSHOTS_TABLE} FINAL
        WHERE ${RESOURCE_FILTER}
          AND time >= now() - INTERVAL 2 MINUTE`,
      params: baseParams,
      schema: resourceSummarySchema,
    });

    return query(args);
  };
}
