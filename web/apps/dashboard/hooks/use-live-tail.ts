import { type QueryKey, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

const LIVE_LOOKBACK_MS = 5 * 60 * 1000;

export type LiveWindow = { startTime: number; endTime: number };

type LiveTailConfig<TRow> = {
  enabled: boolean;
  intervalMs: number;
  queryKey: QueryKey;
  windowStart: number;
  fetchNewest: (window: LiveWindow, signal: AbortSignal | undefined) => Promise<TRow[]>;
  keyOf: (row: TRow) => string;
  historical: readonly TRow[];
};

// Polls the newest rows of the last few minutes, not rows after the newest seen:
// log writers flush late, so older rows can land after newer ones are shown.
export function useLiveTail<TRow>({
  enabled,
  intervalMs,
  queryKey,
  windowStart,
  fetchNewest,
  keyOf,
  historical,
}: LiveTailConfig<TRow>): TRow[] {
  const live = useQuery({
    queryKey: ["live-tail", ...queryKey],
    queryFn: ({ signal }) => {
      const endTime = Date.now();
      return fetchNewest(
        { startTime: Math.max(windowStart, endTime - LIVE_LOOKBACK_MS), endTime },
        signal,
      );
    },
    enabled,
    refetchInterval: intervalMs,
    staleTime: 0,
  });

  const historicalKeys = useMemo(() => new Set(historical.map(keyOf)), [historical, keyOf]);

  return useMemo(
    () => (enabled ? (live.data ?? []).filter((row) => !historicalKeys.has(keyOf(row))) : []),
    [enabled, live.data, historicalKeys, keyOf],
  );
}
