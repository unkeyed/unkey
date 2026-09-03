const dayMs = 24 * 60 * 60 * 1000;
const hourMs = 60 * 60 * 1000;
const bucketMs = 5 * 60 * 1000;

export function alertTimeseriesRange({
  firedAt,
  resolvedAt,
  now,
}: {
  firedAt: number;
  resolvedAt: number | null;
  now: number;
}) {
  const endLimit = Math.min(now, (resolvedAt ?? now) + hourMs);
  return {
    startMs: Math.ceil((firedAt - dayMs) / bucketMs) * bucketMs,
    endMs: Math.floor(endLimit / bucketMs) * bucketMs,
  };
}
