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
  trailingCpuSeconds: z.number(),
  trailingMemoryGiBHours: z.number(),
  trailingDiskGiBHours: z.number(),
  trailingEgressGiB: z.number(),
});

export type DeployMeterUsage = z.infer<typeof deployMeterUsage>;

/**
 * Workspace-level billable Deploy usage for a time window, summed across all
 * resources: the same checkpoint-pair aggregation the hourly billing push in
 * svc/ctrl runs (pkg/clickhouse GetInstanceMeterUsage), so the dashboard shows
 * the numbers that are actually billed. The trailing window is aggregated from
 * the same checkpoint scan so projecting usage does not reread the raw table.
 * Display-only; Stripe meter events remain the billing write path. The query
 * reads the physical table with FINAL because some ClickHouse versions do not
 * propagate the equivalent view's primary-key order and force the window input
 * through a redundant full sort.
 */
export function getDeployMeterUsage(ch: Querier) {
  const query = ch.query({
    query: `
      SELECT
        sumIf(cpu_usec_delta, ts >= {periodStart: Int64}) / 1e6 AS cpuSeconds,
        sumIf(memory_byte_ms, ts >= {periodStart: Int64}) / 1000 / 3600 / pow(1024, 3) AS memoryGiBHours,
        sumIf(disk_byte_ms, ts >= {periodStart: Int64}) / 1000 / 3600 / pow(1024, 3) AS diskGiBHours,
        sumIf(egress_bytes_delta, ts >= {periodStart: Int64}) / pow(1024, 3) AS egressGiB,
        sumIf(cpu_usec_delta, ts >= {trailingStart: Int64}) / 1e6 AS trailingCpuSeconds,
        sumIf(memory_byte_ms, ts >= {trailingStart: Int64}) / 1000 / 3600 / pow(1024, 3) AS trailingMemoryGiBHours,
        sumIf(disk_byte_ms, ts >= {trailingStart: Int64}) / 1000 / 3600 / pow(1024, 3) AS trailingDiskGiBHours,
        sumIf(egress_bytes_delta, ts >= {trailingStart: Int64}) / pow(1024, 3) AS trailingEgressGiB
      FROM (
        SELECT
          ts,
          leadInFrame(ts) OVER w - ts AS dt,
          greatest(0, leadInFrame(cpu_usage_usec) OVER w - cpu_usage_usec) AS cpu_usec_delta,
          if(
            ifNull(attributes.network_attached::Nullable(Bool), false)
            AND leadInFrame(ifNull(attributes.network_attached::Nullable(Bool), false)) OVER w,
            greatest(0, leadInFrame(network_egress_public_bytes) OVER w - network_egress_public_bytes),
            0
          ) AS egress_bytes_delta,
          toFloat64(least(memory_bytes, leadInFrame(memory_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS memory_byte_ms,
          toFloat64(least(disk_allocated_bytes, leadInFrame(disk_allocated_bytes) OVER w)) * toFloat64(leadInFrame(ts) OVER w - ts) AS disk_byte_ms
        FROM default.instance_checkpoints_v1 FINAL
        WHERE ts >= {scanStart: Int64}
          AND ts < {end: Int64}
          AND workspace_id = {workspaceId: String}
        WINDOW w AS (
          PARTITION BY workspace_id, container_uid
          ORDER BY ts, event_kind
          ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
        )
      )
      WHERE dt > 0
        AND dt <= {maxGapMs: Int64}
      SETTINGS
        do_not_merge_across_partitions_select_final = 1,
        max_final_threads = 1,
        optimize_read_in_order = 1,
        optimize_read_in_window_order = 1
    `,
    params: z.object({
      workspaceId: z.string(),
      scanStart: z.int(),
      periodStart: z.int(),
      trailingStart: z.int(),
      end: z.int(),
      maxGapMs: z.int(),
    }),
    schema: deployMeterUsage,
  });

  return async (args: {
    workspaceId: string;
    /** Inclusive lower bound of the billing period, unix millis. */
    periodStart: number;
    /** Inclusive lower bound of the projection window, unix millis. */
    trailingStart: number;
    /** Exclusive upper bound, unix millis. */
    end: number;
  }): Promise<DeployMeterUsage> => {
    const result = await query({
      ...args,
      scanStart: Math.min(args.periodStart, args.trailingStart),
      maxGapMs: MAX_SAMPLE_GAP_MS,
    });
    if (result.err) {
      throw new Error(`Failed to query deploy meter usage: ${result.err.message}`);
    }
    return (
      result.val.at(0) ?? {
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
        trailingCpuSeconds: 0,
        trailingMemoryGiBHours: 0,
        trailingDiskGiBHours: 0,
        trailingEgressGiB: 0,
      }
    );
  };
}

export const deployUsageByScope = z.object({
  projectId: z.string(),
  appId: z.string(),
  environmentId: z.string(),
  cpuSeconds: z.number(),
  memoryGiBHours: z.number(),
  diskGiBHours: z.number(),
  egressGiB: z.number(),
  samplePairs: z.number(),
});

export type DeployUsageByScope = z.infer<typeof deployUsageByScope>;

/**
 * Deploy usage for a window, grouped by project / app / environment, read from
 * the hourly rollup instance_usage_per_hour_v1 (dashboards only; billing stays
 * on the raw checkpoints via GetInstanceMeterUsage).
 *
 * FINAL is mandatory, not an optimisation. The table is a
 * ReplacingMergeTree(computed_at) that the refresh views APPEND to over
 * overlapping windows, so before a merge the same hour exists several times.
 * Summing without FINAL double-counts those generations.
 *
 * Egress counts public egress only, the same meter getDeployMeterUsage bills;
 * private and ingress bytes are recorded in the rollup but not charged.
 */
export function getDeployUsageByScope(ch: Querier) {
  const query = ch.query({
    query: `
      SELECT
        project_id AS projectId,
        app_id AS appId,
        environment_id AS environmentId,
        sum(cpu_seconds) AS cpuSeconds,
        sum(memory_gib_hours) AS memoryGiBHours,
        sum(disk_gib_hours) AS diskGiBHours,
        sum(network_egress_public_bytes) / pow(1024, 3) AS egressGiB,
        toInt64(sum(sample_pairs)) AS samplePairs
      FROM default.instance_usage_per_hour_v1 FINAL
      WHERE workspace_id = {workspaceId: String}
        AND time >= toDateTime(fromUnixTimestamp64Milli({periodStart: Int64}))
        AND time < toDateTime(fromUnixTimestamp64Milli({end: Int64}))
      GROUP BY project_id, app_id, environment_id
      ORDER BY cpuSeconds DESC, projectId, appId, environmentId
      SETTINGS do_not_merge_across_partitions_select_final = 1
    `,
    params: z.object({
      workspaceId: z.string(),
      periodStart: z.int(),
      end: z.int(),
    }),
    schema: deployUsageByScope,
  });

  return async (args: {
    workspaceId: string;
    /** Inclusive lower bound of the billing period, unix millis. */
    periodStart: number;
    /** Exclusive upper bound, unix millis. */
    end: number;
  }): Promise<DeployUsageByScope[]> => {
    const result = await query(args);
    if (result.err) {
      throw new Error(`Failed to query deploy usage by scope: ${result.err.message}`);
    }
    return result.val;
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
    /** Calendar year of the billing month. */
    year: number;
    /** Calendar month, 1-12. */
    month: number;
  }): Promise<ActiveKeysUsage> => {
    const query = ch.query({
      query: `
        SELECT toInt64(uniqExact(key_id)) AS activeKeys
        FROM default.key_verifications_per_month_v3
        WHERE time = makeDate({year: Int32}, {month: Int32}, 1)
          AND source = 'gateway'
          AND workspace_id = {workspaceId: String}
      `,
      params: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),
      schema: activeKeysUsage,
    });

    const result = await query({
      workspaceId: args.workspaceId,
      year: args.year,
      month: args.month,
    });
    if (result.err) {
      throw new Error(`Failed to fetch active keys usage: ${result.err.message}`);
    }
    return result.val.at(0) ?? { activeKeys: 0 };
  };
}
