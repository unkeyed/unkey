import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useFilters } from "../../../hooks/use-filters";
import type { RatelimitQueryLogsPayload } from "../query-logs.schema";

type UseLogsQueryParams = {
  limit?: number;
  pollIntervalMs?: number;
  startPolling?: boolean;
  namespaceId: string;
};

const REALTIME_DATA_LIMIT = 100;

export function useRatelimitLogsQuery({
  namespaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseLogsQueryParams) {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const [historicalLogsMap, setHistoricalLogsMap] = useState(() => new Map<string, RatelimitLog>());
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, RatelimitLog>());

  const [totalCount, setTotalCount] = useState(0);

  const { queryTime: timestamp } = useQueryTime();

  const { filters } = useFilters();

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(realtimeLogsMap.values()));
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const queryParams = useMemo(() => {
    const params: RatelimitQueryLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      requestIds: { filters: [] },
      identifiers: { filters: [] },
      status: { filters: [] },
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
  }, [filters, limit, namespaceId, timestamp]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = useInfiniteQuery(
    trpc.ratelimit.logs.query.infiniteQueryOptions(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }),
  );

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.fetchQuery(
        trpc.ratelimit.logs.query.queryOptions({
          ...queryParams,
          startTime: latestTime ?? Date.now() - pollIntervalMs,
          endTime: Date.now(),
        }),
      );

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

          // Remove oldest entries when exceeding the size limit to prevent memory issues
          // We use min(limit, REALTIME_DATA_LIMIT) to ensure a reasonable upper bound
          if (newMap.size > Math.min(limit, REALTIME_DATA_LIMIT)) {
            // Find and remove the entry with the oldest timestamp
            const entries = Array.from(newMap.entries());
            const oldestEntry = entries.reduce((oldest, current) => {
              return oldest[1].time < current[1].time ? oldest : current;
            });
            newMap.delete(oldestEntry[0]);
          }
        }

        // If nothing was added, return old map to prevent re-render
        return added > 0 ? newMap : prevMap;
      });
    } catch (error) {
      console.error("Error polling for new logs:", error);
    }
  }, [
    queryParams,
    queryClient,
    trpc,
    limit,
    pollIntervalMs,
    historicalLogsMap,
    realtimeLogs,
    historicalLogs,
  ]);

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
      if (initialData.pages.length > 0) {
        setTotalCount(initialData.pages[0].total);
      }
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
    totalCount,
  };
}

const sortLogs = (logs: RatelimitLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
