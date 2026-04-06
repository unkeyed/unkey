import { z } from "zod";
import type { Querier } from "./client";

// ─── Time window config ────────────────────────────────────────────────
//
// Each window picks the cheapest MV that can serve the requested range
// without producing an unreadable pixel-soup chart (target ~60-1500
// buckets per chart). Shorter windows read the per-15s MV for granularity;
// multi-day reads per-minute; weekly reads per-hour; monthly+ read per-day
// or per-month.

export const TIME_WINDOWS = [
  "15m",
  "1h",
  "3h",
  "6h",
  "12h",
  "1d",
  "1w",
  "30d",
  "90d",
  "1y",
] as const;
export type TimeWindow = (typeof TIME_WINDOWS)[number];

type WindowSpec = {
  windowSeconds: number;
  bucketSeconds: number;
  mvTable: string;
};

const WINDOW_CONFIG: Record<TimeWindow, WindowSpec> = {
  "15m": {
    windowSeconds: 15 * 60,
    bucketSeconds: 15,
    mvTable: "default.instance_resources_per_15s_v1",
  },
  "1h": {
    windowSeconds: 60 * 60,
    bucketSeconds: 15,
    mvTable: "default.instance_resources_per_15s_v1",
  },
  "3h": {
    windowSeconds: 3 * 60 * 60,
    bucketSeconds: 60,
    mvTable: "default.instance_resources_per_minute_v1",
  },
  "6h": {
    windowSeconds: 6 * 60 * 60,
    bucketSeconds: 60,
    mvTable: "default.instance_resources_per_minute_v1",
  },
  "12h": {
    windowSeconds: 12 * 60 * 60,
    bucketSeconds: 60,
    mvTable: "default.instance_resources_per_minute_v1",
  },
  "1d": {
    windowSeconds: 24 * 60 * 60,
    bucketSeconds: 60,
    mvTable: "default.instance_resources_per_minute_v1",
  },
  "1w": {
    windowSeconds: 7 * 24 * 60 * 60,
    bucketSeconds: 60 * 60,
    mvTable: "default.instance_resources_per_hour_v1",
  },
  // 30d @ hour buckets = 720 points — still readable, more detail than day
  // buckets would give. per-hour MV TTL is 90 days so the full 30-day
  // window fits.
  "30d": {
    windowSeconds: 30 * 24 * 60 * 60,
    bucketSeconds: 60 * 60,
    mvTable: "default.instance_resources_per_hour_v1",
  },
  // 90d @ day buckets = 90 points. per-hour MV TTL is exactly 90 days, so
  // we'd risk missing the earliest bucket — per-day (365d TTL) is the
  // safer read at this range.
  "90d": {
    windowSeconds: 90 * 24 * 60 * 60,
    bucketSeconds: 24 * 60 * 60,
    mvTable: "default.instance_resources_per_day_v1",
  },
  // 1y @ day buckets = 365 points — dense but readable. per-day MV TTL is
  // 365 days, so the edge is covered. Month buckets would only give 12
  // points, losing too much detail for day-level hover / annotations.
  "1y": {
    windowSeconds: 365 * 24 * 60 * 60,
    bucketSeconds: 24 * 60 * 60,
    mvTable: "default.instance_resources_per_day_v1",
  },
};

// Raw FINAL view for the "live tip" of charts (current, still-filling bucket).
const CHECKPOINTS_VIEW = "default.instance_checkpoints";

// `instance_id` in the checkpoints schema is the k8s pod name. Filtering by
// it scopes results to a single replica (e.g., the instance the user clicked
// in the network panel) instead of summing across all of the deployment's
// pods. Empty string is the "no filter" sentinel because all real pod names
// have at least one character — keeps the WHERE clause stable so the query
// plan caches.
const RESOURCE_FILTER = `
  workspace_id = {workspaceId: String}
  AND resource_type = {resourceType: String}
  AND resource_id = {resourceId: String}
  AND ({instanceName: String} = '' OR instance_id = {instanceName: String})`;

const baseParams = z.object({
  workspaceId: z.string(),
  resourceType: z.enum(["deployment", "sentinel"]),
  resourceId: z.string(),
  instanceName: z.string().default(""),
  window: z.enum(TIME_WINDOWS).default("1h"),
});

const timeseriesPointSchema = z.object({
  x: z.number().int(),
  y: z.number(),
});

// ─── Hybrid timeseries (MV for cold tail + raw for live bucket) ───────
//
// Chart data comes from two places stitched with UNION ALL:
//   - Cold tail: the pre-computed MV for all closed buckets in the window.
//   - Live tip: raw instance_checkpoints for the last MV_GRACE_SECONDS of
//     time, grouped by bucket. Covers the still-filling bucket plus any
//     recently-closed buckets the MV hasn't caught up to yet (ClickHouse
//     MVs fire on INSERT, so under an ingest burst they can lag the raw
//     table by a few seconds; without the grace window, dashboards go
//     flat during catch-up).
//
// The cutoff is `now() - MV_GRACE_SECONDS`, rounded down to the bucket
// boundary. MV returns everything strictly before the cutoff; raw
// returns everything from the cutoff onward, bucketed the same way the
// MV would be. Because the cutoff is bucket-aligned, each bucket is
// served by exactly one side — no double-count.
//
// The outer WHERE y > 0 drops empty tip buckets so the chart doesn't
// show a phantom "0" bar at the right edge before the first sample of
// the current bucket arrives.
const MV_GRACE_SECONDS = 30;

function buildParams() {
  return baseParams.extend({
    windowSeconds: z.number(),
    bucketSeconds: z.number(),
    graceSeconds: z.number(),
    tSec: z.number(),
    windowStartSec: z.number(),
    windowEndSec: z.number(),
    tMs: z.number(),
    windowStartMs: z.number(),
    windowEndMs: z.number(),
  });
}

function specFor(args: z.infer<typeof baseParams>) {
  const spec = WINDOW_CONFIG[args.window];
  // Compute every time boundary the query needs in JS, not as a SQL CTE.
  // ClickHouse's WITH FILL FROM/TO requires a non-Nullable constant;
  // pulling the bound from a subquery returns Nullable(Int64) and the
  // server rejects the query with "Sort FILL FROM expression must be
  // constant with numeric type". Pinning now() once here also keeps the
  // cold/live cutoff aligned with the FILL bounds (the original reason
  // the CTE existed). Clock skew between dashboard host and ClickHouse
  // is sub-second, invisible at any of the chart resolutions.
  const nowSec = Math.floor(Date.now() / 1000);
  const bucket = (sec: number) => Math.floor(sec / spec.bucketSeconds) * spec.bucketSeconds;
  const tSec = bucket(nowSec - MV_GRACE_SECONDS);
  const windowStartSec = bucket(nowSec - spec.windowSeconds);
  const windowEndSec = bucket(nowSec);
  return {
    ...args,
    windowSeconds: spec.windowSeconds,
    bucketSeconds: spec.bucketSeconds,
    graceSeconds: MV_GRACE_SECONDS,
    tSec,
    windowStartSec,
    windowEndSec,
    tMs: tSec * 1000,
    windowStartMs: windowStartSec * 1000,
    windowEndMs: windowEndSec * 1000,
    mvTable: spec.mvTable,
  };
}

// ─── Metric registry ──────────────────────────────────────────────────
//
// Each metric describes how to compute a per-bucket y-value from the MV
// (cold tail) and from raw checkpoints (live tip). Two shapes:
//
//   - Container-reduce-then-aggregate (cpu, memory, disk): the live-tip
//     query first reduces raw rows per container_uid via `perContainer`,
//     then sums across containers via `rawAgg` (which references the
//     `container_value` alias produced by the inner subquery).
//
//   - Direct aggregation (instances): the live-tip query aggregates
//     directly over raw rows. `perContainer` is omitted.
//
// Adding a metric = one entry below. The hybrid query structure, MV-vs-
// raw stitching, parameter binding, and zero-bar suppression are shared.

type Metric = {
  // Per-bucket aggregate over the MV's pre-aggregated columns.
  // Resulting column is aliased AS y by the builder.
  mvAgg: string;
  // Per-bucket aggregate over either raw checkpoints or the per-container
  // subquery output. Aliased AS y by the builder.
  rawAgg: string;
  // Optional: per-container reduction inside the live-tip subquery.
  // Result is aliased AS container_value, available to rawAgg.
  perContainer?: string;
};

const METRICS: Record<string, Metric> = {
  // CPU: cumulative usec counter delta per container (= microseconds the
  // container actually spent on-CPU during the bucket), summed across
  // containers, divided by bucket length and 1000 → millicores.
  cpu: {
    mvAgg: "sum(cpu_usage_usec_max - cpu_usage_usec_min) / {bucketSeconds: UInt32} / 1000",
    perContainer: "max(cpu_usage_usec) - min(cpu_usage_usec)",
    rawAgg: "sum(container_value) / {bucketSeconds: UInt32} / 1000",
  },
  // Memory: peak (working-set) bytes per container during the bucket,
  // summed across containers. max() is monotone so retries don't double-count.
  memory: {
    mvAgg: "sum(memory_bytes_max)",
    perContainer: "max(memory_bytes)",
    rawAgg: "sum(container_value)",
  },
  // Disk: latest used bytes per container during the bucket, summed
  // across containers. Same shape as memory.
  disk: {
    mvAgg: "sum(disk_used_bytes_max)",
    perContainer: "max(disk_used_bytes)",
    rawAgg: "sum(container_value)",
  },
  // Active instances: distinct container_uid count per bucket. No
  // per-container reduce — uniq is already a direct aggregation.
  instances: {
    mvAgg: "uniq(container_uid)",
    rawAgg: "uniq(container_uid)",
  },
  // Network egress: sum of public + private egress bytes counter delta per
  // bucket (max - min, monotone like cpu_usage_usec), divided by bucket
  // length to give bytes/sec. Counters come from the cgroup_skb eBPF
  // collector. v1 sums public + private into one rate; we can split into
  // two stacked colours later if customers need to see the breakdown.
  network_egress: {
    mvAgg:
      "sum((network_egress_public_bytes_max - network_egress_public_bytes_min) + (network_egress_private_bytes_max - network_egress_private_bytes_min)) / {bucketSeconds: UInt32}",
    perContainer:
      "(max(network_egress_public_bytes) - min(network_egress_public_bytes)) + (max(network_egress_private_bytes) - min(network_egress_private_bytes))",
    rawAgg: "sum(container_value) / {bucketSeconds: UInt32}",
  },
  network_ingress: {
    mvAgg:
      "sum((network_ingress_public_bytes_max - network_ingress_public_bytes_min) + (network_ingress_private_bytes_max - network_ingress_private_bytes_min)) / {bucketSeconds: UInt32}",
    perContainer:
      "(max(network_ingress_public_bytes) - min(network_ingress_public_bytes)) + (max(network_ingress_private_bytes) - min(network_ingress_private_bytes))",
    rawAgg: "sum(container_value) / {bucketSeconds: UInt32}",
  },
};

function buildHybridQuery(metric: Metric, mvTable: string): string {
  // All time bounds (cold/live cutoff, MV window start, WITH FILL FROM/TO)
  // are computed in specFor() and passed as scalar parameters. This
  // sidesteps two ClickHouse limits: (1) WITH FILL FROM/TO must be a
  // non-Nullable constant, and a CTE-derived value comes out as
  // Nullable(Int64); (2) `now()` evaluated inline at multiple call sites
  // can resolve to different milliseconds, putting the cold/live cutoff
  // and the FILL bounds in different buckets. Pinning everything to a
  // single client-side timestamp avoids both issues.

  // Live-tip bucketing. Cast ts (DateTime64 millis) down to DateTime seconds
  // before bucketing so the resulting `time` column has the same type as the
  // MV's `time` — otherwise the UNION promotes to DateTime64 and the
  // DateTime-typed WITH FILL bounds below would fail the "types must match
  // exactly" check.
  const tipBucket =
    "toStartOfInterval(toDateTime(intDiv(ts, 1000)), INTERVAL {bucketSeconds: UInt32} SECOND)";

  const liveTip = metric.perContainer
    ? `SELECT time, ${metric.rawAgg} AS y
       FROM (
         SELECT
           container_uid,
           ${tipBucket} AS time,
           ${metric.perContainer} AS container_value
         FROM ${CHECKPOINTS_VIEW}
         WHERE ${RESOURCE_FILTER}
           AND ts >= {tMs: Int64}
         GROUP BY container_uid, time
       )
       GROUP BY time`
    : `SELECT
         ${tipBucket} AS time,
         ${metric.rawAgg} AS y
       FROM ${CHECKPOINTS_VIEW}
       WHERE ${RESOURCE_FILTER}
         AND ts >= {tMs: Int64}
       GROUP BY time`;

  // ORDER BY time WITH FILL FROM/TO pads every missing bucket across the
  // full window with zero-valued rows, so the chart always has a sample
  // at every grid position from `now - windowSeconds` to `now`. The
  // client keeps the overall axis-width decision (anchor to the window
  // vs contract to data extent): it counts non-zero samples to detect
  // sparse data and contracts the x-axis domain accordingly, so this
  // full-window padding doesn't cause the "23h of flat zeros" problem
  // on a new deployment that picked "Past day".
  // WITH FILL fills the ORDER BY column with interpolated values for
  // missing buckets; every OTHER column in the SELECT gets the Int64/
  // Float64 zero default for those filled rows. That matters here
  // because the chart's x-axis reads `originalTimestamp` (the alias `x`)
  // — if we order by `time` and project `x = toUnixTimestamp(time)`,
  // filled rows arrive with x=0 (Unix epoch) while time is the correct
  // bucket, and the chart renders a phantom line from x=0 through the
  // real data. Fix: order by and fill on `x` directly (Int64 millis),
  // so filled rows get the correct millis in the column the chart
  // actually reads.
  return `
    SELECT x, toFloat64(y) AS y
    FROM (
      SELECT
        toInt64(toUnixTimestamp(time) * 1000) AS x,
        ${metric.mvAgg} AS y
      FROM ${mvTable}
      WHERE ${RESOURCE_FILTER}
        AND time >= toDateTime({windowStartSec: UInt32})
        AND time <  toDateTime({tSec: UInt32})
      GROUP BY time

      UNION ALL

      SELECT toInt64(toUnixTimestamp(time) * 1000) AS x, y FROM (${liveTip}) WHERE y > 0
    )
    ORDER BY x ASC
    WITH FILL
      FROM {windowStartMs: Int64}
      TO {windowEndMs: Int64}
      STEP toInt64({bucketSeconds: UInt32} * 1000)`;
}

function makeTimeseriesQuery(metric: Metric) {
  return (ch: Querier) => async (args: z.infer<typeof baseParams>) => {
    const { mvTable, ...params } = specFor(args);
    const query = ch.query({
      query: buildHybridQuery(metric, mvTable),
      params: buildParams(),
      schema: timeseriesPointSchema,
    });
    return query(params);
  };
}

export const getResourceCpuTimeseries = makeTimeseriesQuery(METRICS.cpu);
export const getResourceMemoryTimeseries = makeTimeseriesQuery(METRICS.memory);
export const getResourceDiskTimeseries = makeTimeseriesQuery(METRICS.disk);
export const getResourceInstanceCountTimeseries = makeTimeseriesQuery(METRICS.instances);
export const getResourceNetworkEgressTimeseries = makeTimeseriesQuery(METRICS.network_egress);
export const getResourceNetworkIngressTimeseries = makeTimeseriesQuery(METRICS.network_ingress);

// ─── Current Resource Summary (latest window from raw FINAL view) ─────
//
// Independent of the chart's time window — always reports "right now" from
// raw data. The panel header should feel live (~seconds) regardless of
// whether the chart is showing the past 15 minutes or the past week.
const RATE_WINDOW_MINUTES = 2;
const LIVE_WINDOW_SECONDS = 30;

const resourceSummarySchema = z.object({
  active_instances: z.number().int(),
  current_cpu_millicores: z.number(),
  current_memory_bytes: z.number().int(),
  current_disk_used_bytes: z.number().int(),
  current_egress_bytes_per_sec: z.number(),
  current_ingress_bytes_per_sec: z.number(),
  cpu_allocated_millicores: z.number(),
  memory_allocated_bytes: z.number(),
});

const summaryParams = z.object({
  workspaceId: z.string(),
  resourceType: z.enum(["deployment", "sentinel"]),
  resourceId: z.string(),
  instanceName: z.string().default(""),
});

export function getResourceSummary(ch: Querier) {
  return async (args: z.infer<typeof summaryParams>) => {
    const query = ch.query({
      query: `
        WITH per_container AS (
          SELECT
            container_uid,
            max(ts) AS last_seen_ts,
            max(cpu_usage_usec) - min(cpu_usage_usec) AS cpu_delta_usec,
            greatest(max(ts) - min(ts), 1) AS span_ms,
            argMax(memory_bytes, ts) AS memory_bytes_latest,
            argMax(disk_used_bytes, ts) AS disk_used_bytes_latest,
            argMax(cpu_allocated_millicores, ts) AS cpu_alloc_latest,
            argMax(memory_allocated_bytes, ts) AS memory_alloc_latest,
            (max(network_egress_public_bytes) - min(network_egress_public_bytes))
              + (max(network_egress_private_bytes) - min(network_egress_private_bytes)) AS egress_delta_bytes,
            (max(network_ingress_public_bytes) - min(network_ingress_public_bytes))
              + (max(network_ingress_private_bytes) - min(network_ingress_private_bytes)) AS ingress_delta_bytes
          FROM ${CHECKPOINTS_VIEW}
          WHERE ${RESOURCE_FILTER}
            AND ts >= toUnixTimestamp(now() - INTERVAL {rateWindowMinutes: UInt8} MINUTE) * 1000
          GROUP BY container_uid
        ),
        live_cutoff AS (
          SELECT toUnixTimestamp(now() - INTERVAL {liveWindowSeconds: UInt16} SECOND) * 1000 AS cutoff_ms
        )
        SELECT
          toUInt32(countIf(last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff))) AS active_instances,
          sumIf(cpu_delta_usec / span_ms, last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS current_cpu_millicores,
          sumIf(memory_bytes_latest, last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS current_memory_bytes,
          sumIf(disk_used_bytes_latest, last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS current_disk_used_bytes,
          sumIf(egress_delta_bytes / (span_ms / 1000), last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS current_egress_bytes_per_sec,
          sumIf(ingress_delta_bytes / (span_ms / 1000), last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS current_ingress_bytes_per_sec,
          sumIf(cpu_alloc_latest, last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS cpu_allocated_millicores,
          sumIf(memory_alloc_latest, last_seen_ts >= (SELECT cutoff_ms FROM live_cutoff)) AS memory_allocated_bytes
        FROM per_container`,
      params: summaryParams.extend({
        rateWindowMinutes: z.number(),
        liveWindowSeconds: z.number(),
      }),
      schema: resourceSummarySchema,
    });

    return query({
      ...args,
      rateWindowMinutes: RATE_WINDOW_MINUTES,
      liveWindowSeconds: LIVE_WINDOW_SECONDS,
    });
  };
}
