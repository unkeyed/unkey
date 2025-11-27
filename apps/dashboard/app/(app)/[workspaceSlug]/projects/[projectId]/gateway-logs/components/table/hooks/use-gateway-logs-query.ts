import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import type { LogsRequestSchema } from "@/lib/schemas/logs.schema";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import type { Log } from "@unkey/clickhouse/src/logs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { EXCLUDED_HOSTS } from "../../../constants";
import { useGatewayLogsFilters } from "../../../hooks/use-gateway-logs-filters";

import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";

// Constants
const REALTIME_DATA_LIMIT = 100;

// Types
type UseGatewayLogsQueryParams = {
  limit?: number;
  pollIntervalMs?: number;
  startPolling?: boolean;
};

const FILTER_FIELD_MAPPING = {
  status: "status",
  methods: "method",
  paths: "path",
  host: "host",
  requestId: "requestId",
} as const;

const TIME_FIELDS = ["startTime", "endTime", "since"] as const;

export function useGatewayLogsQuery({
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseGatewayLogsQueryParams = {}) {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const [historicalLogsMap, setHistoricalLogsMap] = useState(() => new Map<string, Log>());
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, Log>());

  const [totalCount, setTotalCount] = useState(0);
  const { filters } = useGatewayLogsFilters();
  const { queryTime: timestamp } = useQueryTime();

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(realtimeLogsMap.values()));
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  // "memo" required for preventing double trpc call during initial render
  const queryParams = useMemo(() => {
    const params: LogsRequestSchema = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      host: { filters: [], exclude: EXCLUDED_HOSTS },
      requestId: { filters: [] },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
      since: "",
    };

    filters.forEach((filter) => {
      const paramKey = FILTER_FIELD_MAPPING[filter.field as keyof typeof FILTER_FIELD_MAPPING];

      if (paramKey && params[paramKey as keyof typeof params]) {
        switch (filter.field) {
          case "status": {
            const statusValue = Number.parseInt(filter.value as string);
            if (Number.isNaN(statusValue)) {
              console.error("Status filter value must be a valid number");
              return;
            }
            params.status?.filters.push({
              operator: "is",
              value: statusValue,
            });
            break;
          }

          case "methods":
          case "host":
          case "requestId": {
            if (typeof filter.value !== "string") {
              console.error(`${filter.field} filter value must be a string`);
              return;
            }
            const targetParam = params[paramKey as keyof typeof params] as {
              filters: { operator: string; value: string }[];
            };
            targetParam.filters.push({
              operator: "is",
              value: filter.value,
            });
            break;
          }

          case "paths": {
            if (typeof filter.value !== "string") {
              console.error("Path filter value must be a string");
              return;
            }
            params.path?.filters.push({
              operator: filter.operator,
              value: filter.value,
            });
            break;
          }
        }
      } else if (TIME_FIELDS.includes(filter.field as (typeof TIME_FIELDS)[number])) {
        switch (filter.field) {
          case "startTime":
          case "endTime": {
            if (typeof filter.value !== "number") {
              console.error(`${filter.field} filter value must be a number`);
              return;
            }
            params[filter.field] = filter.value;
            break;
          }
          case "since": {
            if (typeof filter.value !== "string") {
              console.error("Since filter value must be a string");
              return;
            }
            params.since = filter.value;
            break;
          }
        }
      }
    });

    return params;
  }, [filters, limit, timestamp]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = useInfiniteQuery(
    trpc.logs.queryLogs.infiniteQueryOptions(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    })
  );

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.fetchQuery(
        trpc.logs.queryLogs.queryOptions({
          ...queryParams,
          startTime: latestTime ?? Date.now() - pollIntervalMs,
          endTime: Date.now(),
        })
      );

      if (result.logs.length === 0) {
        return;
      }

      setRealtimeLogsMap((prevMap) => {
        const newMap = new Map(prevMap);
        let added = 0;

        for (const log of result.logs) {
          // Skip if exists in either map
          if (newMap.has(log.request_id) || historicalLogsMap.has(log.request_id)) {
            continue;
          }

          newMap.set(log.request_id, log);
          added++;

          // Remove oldest entries when exceeding the size limit `100`
          if (newMap.size > Math.min(limit, REALTIME_DATA_LIMIT)) {
            const entries = Array.from(newMap.entries());
            const oldestEntry = entries.reduce((oldest, current) => {
              return oldest[1].time < current[1].time ? oldest : current;
            });
            newMap.delete(oldestEntry[0]);
          }
        }

        return added > 0 ? newMap : prevMap;
      });
    } catch (error) {
      console.error("Error polling for new logs:", error);
    }
  }, [
    queryParams,
    queryClient,
    trpc.logs.queryLogs,
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
      const newMap = new Map<string, Log>();
      initialData.pages.forEach((page) => {
        page.logs.forEach((log) => {
          newMap.set(log.request_id, log);
        });
      });
      setHistoricalLogsMap(newMap);

      if (initialData.pages.length > 0) {
        setTotalCount(initialData.pages[0].total);
      }
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
    total: totalCount,
  };
}

const sortLogs = (logs: Log[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
