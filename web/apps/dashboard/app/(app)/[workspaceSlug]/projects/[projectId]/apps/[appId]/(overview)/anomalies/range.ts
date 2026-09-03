export type AnomalyRangePreset = "24h" | "6h" | "7d";

export const fiveMinutesMs = 5 * 60 * 1000;
export const hourMs = 60 * 60 * 1000;
export const dayMs = 24 * hourMs;

const presetDurationMs = {
  "6h": 6 * hourMs,
  "24h": dayMs,
  "7d": 7 * dayMs,
} satisfies Record<AnomalyRangePreset, number>;

export function lastClosedBucketEnd(now: number, bucketMs = fiveMinutesMs): number {
  return Math.floor(now / bucketMs) * bucketMs;
}

export function presetRange(preset: AnomalyRangePreset, now: number) {
  const endMs = lastClosedBucketEnd(now);
  return { startMs: endMs - presetDurationMs[preset], endMs };
}

export function focusedAlertRange(windowStart: number, windowEnd: number, now: number) {
  const endMs = lastClosedBucketEnd(now);
  return {
    startMs: Math.max(0, windowStart - 3 * hourMs),
    endMs: Math.min(endMs, windowEnd + 3 * hourMs),
  };
}

export function formatRangeDateTime(value: number): string {
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(value);
}
