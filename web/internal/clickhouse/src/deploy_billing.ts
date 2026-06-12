import { z } from "zod";
import type { Querier } from "./client";

/**
 * Longest interval between two consecutive checkpoints that still counts as
 * continuous usage. Mirrors maxSampleGapMillis in pkg/clickhouse, the
 * authoritative billing query: a larger gap means the agent was down and we
 * drop the interval rather than guess, under-counting instead of over-charging.
 */
const MAX_SAMPLE_GAP_MS = 2 * 60 * 1000;

export const deployMeterUsage = z.object({
  cpuSeconds: z.number(),
  memoryGiBHours: z.number(),
  diskGiBHours: z.number(),
  egressGiB: z.number(),
});

export type DeployMeterUsage = z.infer<typeof deployMeterUsage>;

/**
 * Workspace-level billable Deploy usage for a time window, summed across all
 * resources: the same checkpoint-pair aggregation the hourly billing push in
 * svc/ctrl runs (pkg/clickhouse GetInstanceMeterUsage), so the dashboard shows
 * the numbers that are actually billed. Display-only; Stripe meter events
 * remain the billing write path.
 */
export function getDeployMeterUsage(ch: Querier) {
  return async (args: {
    workspaceId: string;
    /** Inclusive lower bound, unix millis. */
    start: number;
    /** Exclusive upper bound, unix millis. */
    end: number;
  }): Promise<DeployMeterUsage> => {
    const query = ch.query({
      query: `
    SELECT
      sum(cpu_usec_delta) / 1e6 AS cpuSeconds,
      sum(memory_byte_ms) / 1000 / 3600 / pow(1024, 3) AS memoryGiBHours,
      sum(disk_byte_ms) / 1000 / 3600 / pow(1024, 3) AS diskGiBHours,
      sum(egress_bytes_delta) / pow(1024, 3) AS egressGiB
    FROM (
      SELECT
        leadInFrame(ts) OVER w - ts AS dt,
        greatest(0, leadInFrame(cpu_usage_usec) OVER w - cpu_usage_usec) AS cpu_usec_delta,
        greatest(0, leadInFrame(network_egress_public_bytes) OVER w - network_egress_public_bytes) AS egress_bytes_delta,
        toFloat64(least(memory_bytes, leadInFrame(memory_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS memory_byte_ms,
        toFloat64(least(disk_allocated_bytes, leadInFrame(disk_allocated_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS disk_byte_ms
      FROM default.instance_checkpoints
      WHERE ts >= {start: Int64}
        AND ts < {end: Int64}
        AND workspace_id = {workspaceId: String}
      WINDOW w AS (
        PARTITION BY workspace_id, container_uid
        ORDER BY ts, event_kind
        ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
      )
    )
    WHERE dt > 0 AND dt <= {maxGapMs: Int64}
    SETTINGS do_not_merge_across_partitions_select_final = 1
    `,
      params: z.object({
        workspaceId: z.string(),
        start: z.int(),
        end: z.int(),
        maxGapMs: z.int(),
      }),
      schema: deployMeterUsage,
    });

    const res = await query({ ...args, maxGapMs: MAX_SAMPLE_GAP_MS });
    if (res.err) {
      throw new Error(`Failed to query deploy meter usage: ${res.err.message}`);
    }
    return (
      res.val.at(0) ?? {
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
      }
    );
  };
}

export const activeKeysUsage = z.object({
  activeKeys: z.number(),
});

export type ActiveKeysUsage = z.infer<typeof activeKeysUsage>;

/**
 * Distinct keys verified through the Deploy gateway (source = 'gateway') in
 * the billing month, regardless of outcome: a RATE_LIMITED or DISABLED
 * verification is still work done for that key. Mirrors GetActiveKeysUsage in
 * pkg/clickhouse, the authoritative billing query. Display-only.
 */
export function getActiveKeysUsage(ch: Querier) {
  return async (args: {
    workspaceId: string;
    /** Any instant inside the billing month, unix millis. */
    month: number;
  }): Promise<ActiveKeysUsage> => {
    const query = ch.query({
      query: `
        SELECT toInt64(uniqExact(key_id)) AS activeKeys
        FROM default.key_verifications_per_month_v3
        WHERE time = toDate(toStartOfMonth(fromUnixTimestamp64Milli({month: Int64})))
          AND source = 'gateway'
          AND workspace_id = {workspaceId: String}
      `,
      params: z.object({ workspaceId: z.string(), month: z.number().int() }),
      schema: activeKeysUsage,
    });

    const result = await query({ workspaceId: args.workspaceId, month: args.month });
    if (result.err) {
      throw new Error(`Failed to fetch active keys usage: ${result.err.message}`);
    }
    return result.val.at(0) ?? { activeKeys: 0 };
  };
}
