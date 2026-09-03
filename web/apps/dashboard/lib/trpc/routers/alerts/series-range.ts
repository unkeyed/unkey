const fiveMinutesMs = 5 * 60 * 1000;
const hourMs = 60 * 60 * 1000;

export function alertSeriesRange({
  startMs,
  endMs,
  resolution,
  now,
}: {
  startMs: number;
  endMs: number;
  resolution: "5m" | "1h";
  now: number;
}) {
  const bucketMs = resolution === "5m" ? fiveMinutesMs : hourMs;
  const lastClosedBucketEnd = Math.floor(now / bucketMs) * bucketMs;
  return {
    startMs: Math.floor(startMs / bucketMs) * bucketMs,
    endMs: Math.min(Math.ceil(endMs / bucketMs) * bucketMs, lastClosedBucketEnd),
  };
}
