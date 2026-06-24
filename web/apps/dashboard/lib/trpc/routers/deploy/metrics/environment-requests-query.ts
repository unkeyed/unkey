import { INTERVAL_MS, type Interval } from "@unkey/clickhouse/src/frontline/environment-requests";

const minuteMs = 60_000;
const hourMs = 60 * minuteMs;
const dayMs = 24 * hourMs;

const maxWindowMs = 7 * dayMs;

// Range is the coarse label shown alongside the chart. It describes the
// displayed window, while interval only controls the ClickHouse aggregate
// resolution used to read that window.
export type Range = "hour" | "day" | "week";

// EnvironmentRequestsQuery is the resolved plan for one request-metrics query:
// the display [Range], the chosen [Interval], its bucket size in ms, and the
// bucket-aligned half-open grid [startTimeMs, endTimeMs) the chart spans.
export type EnvironmentRequestsQuery = {
  range: Range;
  interval: Interval;
  bucketMs: number;
  startTimeMs: number;
  endTimeMs: number;
};

// selectEnvironmentRequestsQuery maps an app's lifetime to the display range,
// bucket interval, and grid its request chart should read. It is the single
// place that classifies app lifetime: younger apps get finer intervals, and the
// lookback is capped at 7 days so the query stays bounded. The interval names
// the resolution; the ClickHouse package resolves it to a table. `now` is a
// parameter rather than read from the clock so the result is deterministic and
// testable.
export function selectEnvironmentRequestsQuery(
  appCreatedAtMs: number,
  now: number = Date.now(),
): EnvironmentRequestsQuery {
  const appLifetimeMs = Math.max(0, now - appCreatedAtMs);
  const startTimeMs = Math.max(appCreatedAtMs, now - maxWindowMs);

  // Floor both bounds to the interval's bucket size so the grid lines up with
  // the aggregate table's bucket-aligned `time` column; WITH FILL needs that
  // alignment or filled buckets land between real rows. endTimeMs floors
  // `now`, so the current in-flight bucket is excluded. bucketMs comes from the
  // ClickHouse package's INTERVAL_MS, so the grid and the query's WITH FILL
  // STEP share one definition of the interval.
  const query = (range: Range, interval: Interval): EnvironmentRequestsQuery => {
    const bucketMs = INTERVAL_MS[interval];
    return {
      range,
      interval,
      bucketMs,
      startTimeMs: Math.floor(startTimeMs / bucketMs) * bucketMs,
      endTimeMs: Math.floor(now / bucketMs) * bucketMs,
    };
  };

  // Intervals are chosen so each window produces around ~168 points, the chart
  // width, so the series renders one point per bucket with no client-side
  // downsampling. The 15m tier runs through 48h (192 points, slightly over) so
  // resolution stays fine across the first two days instead of dropping at 24h.
  // Bucket counts at each tier's upper edge: 1h@1m=60, 12h@5m=144, 24h@15m=96,
  // 48h@15m=192, 7d@1h=168.
  if (appLifetimeMs < hourMs) {
    return query("hour", "1m");
  }
  if (appLifetimeMs < 12 * hourMs) {
    return query("day", "5m");
  }
  if (appLifetimeMs <= dayMs) {
    return query("day", "15m");
  }
  if (appLifetimeMs <= 2 * dayMs) {
    return query("week", "15m");
  }
  return query("week", "1h");
}
