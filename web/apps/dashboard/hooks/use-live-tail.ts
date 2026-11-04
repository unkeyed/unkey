import { type QueryKey, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

type LiveTailConfig<TRow> = {
  enabled: boolean;
  intervalMs: number;
  queryKey: QueryKey;
  fetchNewest: (signal: AbortSignal | undefined) => Promise<TRow[]>;
  keyOf: (row: TRow) => string;
  historical: readonly TRow[];
};

// Polls the newest rows of the whole window, not rows after the newest seen:
// log writers flush late, so older rows can land after newer ones are shown.
export function useLiveTail<TRow>({
  enabled,
  intervalMs,
  queryKey,
  fetchNewest,
  keyOf,
  historical,
}: LiveTailConfig<TRow>): TRow[] {
  const live = useQuery({
    queryKey: ["live-tail", ...queryKey],
    queryFn: ({ signal }) => fetchNewest(signal),
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
