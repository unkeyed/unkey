import { trpc } from "@/lib/trpc/client";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { useCallback, useEffect, useMemo, useState } from "react";
import { HISTORICAL_DATA_WINDOW } from "../../../constants";
import { useFilters } from "../../../hooks/use-filters";
import type { RatelimitQueryLogsPayload } from "../query-logs.schema";

// Duration in milliseconds for historical data fetch window (12 hours)
type UseLogsQueryParams = {
  limit?: number;
  pollIntervalMs?: number;
  startPolling?: boolean;
  namespaceId?: string;
};

export function useRatelimitLogsQuery({
  namespaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseLogsQueryParams = {}) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(() => new Map<string, RatelimitLog>());
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, RatelimitLog>());

  const { filters } = useFilters();
  const queryClient = trpc.useUtils();

  const realtimeLogs = useMemo(() => {
    return Array.from(realtimeLogsMap.values());
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  //Required for preventing double trpc call during initial render
  const dateNow = useMemo(() => Date.now(), []);
  const queryParams = useMemo(() => {
    const params: RatelimitQueryLogsPayload = {
      limit,
      startTime: dateNow - HISTORICAL_DATA_WINDOW,
      endTime: dateNow,
      requestIds: { filters: [] },
      identifiers: { filters: [] },
      status: { filters: [] },
      //@ts-expect-error will be fixed later
      namespaceId,
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "identifiers": {
          if (typeof filter.value !== "string") {
            console.error("Identifiers filter value type has to be 'string'");
            return;
          }
          params.identifiers?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "requestIds": {
          if (typeof filter.value !== "string") {
            console.error("Request ID filter value type has to be 'string'");
            return;
          }
          params.requestIds?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }

        case "status": {
          if (typeof filter.value !== "string") {
            console.error("Status filter value type has to be 'string'");
            return;
          }
          params.status?.filters.push({
            operator: "is",
            value: filter.value as "blocked" | "passed",
          });
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'string'`);
            return;
          }
          params[filter.field] = filter.value;
          break;
        }

        case "since": {
          if (typeof filter.value !== "string") {
            console.error("Since filter value type has to be 'string'");
            return;
          }
          params.since = filter.value;
          break;
        }
      }
    });

    return params;
  }, [filters, limit, dateNow, namespaceId]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.ratelimit.logs.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialCursor: { requestId: null, time: null },
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Query for new logs (polling)
  // biome-ignore lint/correctness/useExhaustiveDependencies: biome wants to everything as dep
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.ratelimit.logs.query.fetch({
        ...queryParams,
        startTime: latestTime ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
      });

      if (result.ratelimitLogs.length === 0) {
        return;
      }

      setRealtimeLogsMap((prevMap) => {
        const newMap = new Map(prevMap);
        let added = 0;

        for (const log of result.ratelimitLogs) {
          // Skip if exists in either map
          if (newMap.has(log.request_id) || historicalLogsMap.has(log.request_id)) {
            continue;
          }

          newMap.set(log.request_id, log);
          added++;

          if (newMap.size > Math.min(limit, 100)) {
            const oldestKey = Array.from(newMap.keys()).shift()!;
            newMap.delete(oldestKey);
          }
        }

        // If nothing was added, return old map to prevent re-render
        return added > 0 ? newMap : prevMap;
      });
    } catch (error) {
      console.error("Error polling for new logs:", error);
    }
  }, [queryParams, queryClient, limit, pollIntervalMs, historicalLogsMap]);

  // Set up polling effect
  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  // Update historical logs effect
  useEffect(() => {
    if (initialData) {
      const newMap = new Map<string, RatelimitLog>();
      initialData.pages.forEach((page) => {
        page.ratelimitLogs.forEach((log) => {
          newMap.set(log.request_id, log);
        });
      });
      setHistoricalLogsMap(newMap);
    }
  }, [initialData]);

  // Reset realtime logs effect
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  return {
    realtimeLogs,
    historicalLogs,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    isPolling: startPolling,
  };
}
