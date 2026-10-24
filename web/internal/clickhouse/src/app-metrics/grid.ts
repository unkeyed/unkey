export const RANGE_PRESETS = ["1h", "6h", "24h", "3d", "7d", "14d", "30d"] as const;
export type RangePreset = (typeof RANGE_PRESETS)[number];

export type Bucket = "1m" | "5m" | "15m" | "1h" | "2h" | "6h";

const MINUTE_MS = 60 * 1000;
const HOUR_MS = 60 * MINUTE_MS;
export const DAY_MS = 24 * HOUR_MS;

export const BUCKET_MS = {
  "1m": MINUTE_MS,
  "5m": 5 * MINUTE_MS,
  "15m": 15 * MINUTE_MS,
  "1h": HOUR_MS,
  "2h": 2 * HOUR_MS,
  "6h": 6 * HOUR_MS,
} satisfies Record<Bucket, number>;

type RangePlan = { windowMs: number; bucket: Bucket; live: boolean };

export const RANGE_PLANS = {
  "1h": { windowMs: HOUR_MS, bucket: "1m", live: true },
  "6h": { windowMs: 6 * HOUR_MS, bucket: "5m", live: true },
  "24h": { windowMs: DAY_MS, bucket: "15m", live: true },
  "3d": { windowMs: 3 * DAY_MS, bucket: "1h", live: false },
  "7d": { windowMs: 7 * DAY_MS, bucket: "1h", live: false },
  "14d": { windowMs: 14 * DAY_MS, bucket: "2h", live: false },
  "30d": { windowMs: 30 * DAY_MS, bucket: "6h", live: false },
} as const satisfies Record<RangePreset, RangePlan>;

// Writers buffer up to 5s before the synchronous MVs see a row, so a bucket is
// only read once this long has passed after it closed.
export const SETTLE_MS = 15_000;

/**
 * Half-open [startMs, endMs) in Unix ms, both multiples of bucketMs (epoch
 * aligned, UTC). endMs is the start of the in-flight bucket as of SETTLE_MS
 * ago, `floor((now - 15s) / bucketMs) * bucketMs`, so no series includes a
 * bucket that writers may still be filling.
 */
export type Grid = {
  range: RangePreset;
  bucket: Bucket;
  bucketMs: number;
  startMs: number;
  endMs: number;
};

export function planGrid(range: RangePreset, endMs: number): Grid {
  const plan = RANGE_PLANS[range];
  const bucketMs = BUCKET_MS[plan.bucket];
  const end = Math.floor(endMs / bucketMs) * bucketMs;
  return {
    range,
    bucket: plan.bucket,
    bucketMs,
    startMs: end - plan.windowMs,
    endMs: end,
  };
}

export function latestGrid(range: RangePreset, nowMs: number): Grid {
  return planGrid(range, nowMs - SETTLE_MS);
}

export function clampedGrid(range: RangePreset, endMs: number, nowMs: number): Grid {
  return planGrid(range, Math.min(endMs, nowMs - SETTLE_MS));
}

export function nextTickMs(grid: Grid): number {
  return grid.endMs + grid.bucketMs + SETTLE_MS;
}

export type Rollup = { table: string; bucketMs: number; ttlMs: number };

export function findRollup<R extends Rollup>(
  rollups: readonly R[],
  grid: Grid,
  nowMs: number,
): R | null {
  return (
    [...rollups]
      .sort((a, b) => b.bucketMs - a.bucketMs)
      .find(
        (r) => grid.bucketMs % r.bucketMs === 0 && nowMs - grid.startMs + grid.bucketMs <= r.ttlMs,
      ) ?? null
  );
}

export function pickRollup<R extends Rollup>(rollups: readonly R[], grid: Grid, nowMs: number): R {
  const rollup = findRollup(rollups, grid, nowMs);
  if (!rollup) {
    throw new Error(`No rollup serves a ${grid.bucket} grid over ${grid.range}`);
  }
  return rollup;
}
