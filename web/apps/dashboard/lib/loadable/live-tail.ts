export const LIVE_LOOKBACK_MS = 5 * 60 * 1000;
export const LIVE_ROW_CAP = 100;

export type LiveWindow = { startTime: number; endTime: number };

export function livePollWindow({
  historyStart,
  historyEnd,
  lastPollAt,
  now,
}: {
  historyStart: number;
  historyEnd: number;
  lastPollAt: number | undefined;
  now: number;
}): LiveWindow {
  const covered = Math.min(lastPollAt ?? historyEnd, now);
  return { startTime: Math.max(historyStart, covered - LIVE_LOOKBACK_MS), endTime: now };
}

export function mergeLiveRows<TRow>(
  previous: readonly TRow[],
  fetched: readonly TRow[],
  { keyOf, timeOf }: { keyOf: (row: TRow) => string; timeOf: (row: TRow) => number },
): TRow[] {
  const byKey = new Map<string, TRow>();
  for (const row of previous) {
    byKey.set(keyOf(row), row);
  }
  for (const row of fetched) {
    byKey.set(keyOf(row), row);
  }
  return [...byKey.values()]
    .sort((a, b) => timeOf(b) - timeOf(a) || keyOf(b).localeCompare(keyOf(a)))
    .slice(0, LIVE_ROW_CAP);
}
