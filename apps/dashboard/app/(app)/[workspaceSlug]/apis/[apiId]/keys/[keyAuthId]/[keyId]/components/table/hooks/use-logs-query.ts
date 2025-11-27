import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useCallback, useEffect, useMemo, useState } from "react";
import { keyDetailsFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { KeyDetailsLogsPayload } from "../query-logs.schema";

import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";

// Maximum number of real-time logs to store
const REALTIME_DATA_LIMIT = 100;

type UseKeyDetailsLogsQueryParams = {
  limit?: number;
  keyId: string;
  keyspaceId: string;
  pollIntervalMs?: number;
  startPolling?: boolean;
};

export function useKeyDetailsLogsQuery({
  keyId,
  keyspaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseKeyDetailsLogsQueryParams) {
  const queryClient = useQueryClient();
  const trpc = useTRPC();
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, KeyDetailsLog>(),
  );
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, KeyDetailsLog>());

  const [totalCount, setTotalCount] = useState(0);
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(realtimeLogsMap.values()));
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  // Combined logs for rendering
  const logs = useMemo(() => {
    // First get all realtime logs
    const combinedLogs = [...realtimeLogs];

    // Then add historical logs that aren't already in realtime
    for (const log of historicalLogs) {
      if (!realtimeLogsMap.has(log.request_id)) {
        combinedLogs.push(log);
      }
    }

    return sortLogs(combinedLogs);
  }, [realtimeLogs, historicalLogs, realtimeLogsMap]);

  const queryParams = useMemo(() => {
    const params: KeyDetailsLogsPayload = {
      limit,
      keyId,
      keyspaceId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      outcomes: [],
      tags: [],
      since: "",
    };

    filters.forEach((filter) => {
      const fieldConfig = keyDetailsFilterFieldConfig[filter.field];
      const validOperators = fieldConfig?.operators;
      if (!validOperators) {
        return;
      }

      switch (filter.field) {
        case "tags": {
          if (typeof filter.value === "string") {
            params.tags?.push({
              value: filter.value,
              operator: filter.operator as "is" | "contains" | "startsWith" | "endsWith",
            });
          }
          break;
        }
        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes?.push({
              value: filter.value as ValidOutcome,
              operator: "is",
            });
          }
          break;
        }
        case "startTime":
        case "endTime": {
          const numValue =
            typeof filter.value === "number"
              ? filter.value
              : typeof filter.value === "string"
                ? Number(filter.value)
                : Number.NaN;
          if (!Number.isNaN(numValue)) {
            params[filter.field] = numValue;
          }
          break;
        }
        case "since":
          if (typeof filter.value === "string") {
            params.since = filter.value;
          }
          break;
      }
    });

    return params;
  }, [filters, limit, timestamp, keyId, keyspaceId]);

  // Main query for historical data
  const {
    data: logData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading,
  } = useInfiniteQuery(
    trpc.key.logs.query.infiniteQueryOptions(queryParams, {
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
        trpc.key.logs.query.queryOptions({
          ...queryParams,
          startTime: latestTime ?? Date.now() - pollIntervalMs,
          endTime: Date.now(),
        }),
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

          // Remove oldest entries when exceeding the size limit
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
      console.error("Error polling for new key details logs:", error);
    }
  }, [
    queryParams,
    queryClient,
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
    if (logData) {
      const newMap = new Map<string, KeyDetailsLog>();
      logData.pages.forEach((page) => {
        page.logs.forEach((log) => {
          newMap.set(log.request_id, log);
        });
      });
      setHistoricalLogsMap(newMap);

      if (logData.pages.length > 0) {
        setTotalCount(logData.pages[0].total);
      }
    }
  }, [logData]);

  // Reset realtime logs effect
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  return {
    logs,
    realtimeLogs,
    historicalLogs,
    totalCount: totalCount || 0,
    isLoading,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    isPolling: startPolling,
  };
}

// Helper function to sort logs by time in descending order (newest first)
const sortLogs = (logs: KeyDetailsLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
