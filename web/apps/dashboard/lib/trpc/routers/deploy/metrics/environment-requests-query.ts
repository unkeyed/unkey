import { INTERVAL_MS, type Interval } from "@unkey/clickhouse/src/frontline/environment-requests";

const minuteMs = 60_000;
const hourMs = 60 * minuteMs;
const dayMs = 24 * hourMs;

const maxWindowMs = 7 * dayMs;

// WINDOW_MS is the display window each interval covers, sized to land near the
// ~168-bucket chart width. The lifetime tiers pick an interval and the window
// follows from it, so the span a chart shows can never drift from its label.
// 5m has no tier today but keeps its canonical window so the map stays a
// complete interval-to-window spec.
const WINDOW_MS = {
  "1m": hourMs, // 60 buckets
  "5m": 12 * hourMs, // 144 buckets
  "15m": dayMs, // 96 buckets
  "1h": maxWindowMs, // 168 buckets
} satisfies Record<Interval, number>;

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

  // The window spans the full tier anchored at `now`, not the app's lifetime, so
  // a brand-new app still gets a full-width chart instead of two or three
  // points. Buckets before the app existed have no rows and come back as zeros
  // via WITH FILL.
  //
  // Floor both bounds to the interval's bucket size so the grid lines up with
  // the aggregate table's bucket-aligned `time` column; WITH FILL needs that
  // alignment or filled buckets land between real rows. endTimeMs floors `now`,
  // so the current in-flight bucket is excluded. bucketMs comes from the
  // ClickHouse package's INTERVAL_MS, so the grid and the query's WITH FILL STEP
  // share one definition of the interval.
  const query = (range: Range, interval: Interval): EnvironmentRequestsQuery => {
    const bucketMs = INTERVAL_MS[interval];
    const windowMs = WINDOW_MS[interval];
    return {
      range,
      interval,
      bucketMs,
      startTimeMs: Math.floor((now - windowMs) / bucketMs) * bucketMs,
      endTimeMs: Math.floor(now / bucketMs) * bucketMs,
    };
  };

  // Tier is chosen by app lifetime, with one range, one interval, and one
  // window each: hour, day, and week never overlap. Each range's window matches
  // its label, so a "this week" header is never shown over a shorter span; once
  // an app is older than a day it reads as a full week of data (mostly
  // zero-filled buckets at first). Each tier produces around ~168 points, the
  // chart width, so the series renders one point per bucket with no client-side
  // downsampling.
  // Bucket counts: 1h@1m=60, 24h@15m=96, 7d@1h=168.
  // The window never exceeds 7 days, the retention of the aggregate tables.
  if (appLifetimeMs < hourMs) {
    return query("hour", "1m");
  }
  if (appLifetimeMs <= dayMs) {
    return query("day", "15m");
  }
  return query("week", "1h");
}
