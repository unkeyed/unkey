import {
  LIVE_ROW_CAP,
  type LiveWindow,
  livePollWindow,
  mergeLiveRows,
} from "@/lib/loadable/live-tail";
import { type QueryKey, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

type LiveTailConfig<TRow extends { time: number }> = {
  enabled: boolean;
  intervalMs: number;
  queryKey: QueryKey;
  history: LiveWindow;
  fetchNewest: (
    request: LiveWindow & { limit: number },
    signal: AbortSignal | undefined,
  ) => Promise<TRow[]>;
  keyOf: (row: TRow) => string;
  historical: readonly TRow[];
};

// Each poll reads only a few minutes past the previous one and merges into the
// rows already shown: log writers flush late, and a full-window read is costly.
export function useLiveTail<TRow extends { time: number }>({
  enabled,
  intervalMs,
  queryKey,
  history,
  fetchNewest,
  keyOf,
  historical,
}: LiveTailConfig<TRow>): TRow[] {
  const queryClient = useQueryClient();
  const liveKey = ["live-tail", ...queryKey];
  const live = useQuery({
    queryKey: liveKey,
    queryFn: async ({ signal }) => {
      const previous = queryClient.getQueryState<TRow[]>(liveKey);
      const window = livePollWindow({
        historyStart: history.startTime,
        historyEnd: history.endTime,
        lastPollAt: previous?.dataUpdatedAt || undefined,
        now: Date.now(),
      });
      const fetched = await fetchNewest({ ...window, limit: LIVE_ROW_CAP }, signal);
      return mergeLiveRows(previous?.data ?? [], fetched, { keyOf, timeOf: (row) => row.time });
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
