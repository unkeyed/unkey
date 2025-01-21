import { trpc } from "@/lib/trpc/client";
import type { Log } from "@unkey/clickhouse/src/logs";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { z } from "zod";
import { useFilters } from "../../../hooks/use-filters";
import type { queryLogsPayload } from "../query-logs.schema";

type UseLogsQueryParams = {
  limit?: number;
  pollIntervalMs?: number;
  startPolling?: boolean;
};

export function useLogsQuery({
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseLogsQueryParams = {}) {
  const [historicalLogs, setHistoricalLogs] = useState<Log[]>([]);
  const [realtimeLogs, setRealtimeLogs] = useState<Log[]>([]);
  const { filters } = useFilters();
  const queryClient = trpc.useUtils();

  const timestamps = useMemo(
    () => ({
      startTime: Date.now() - 24 * 60 * 60 * 1000,
      endTime: Date.now(),
    }),
    [],
  );

  const queryParams = useMemo(() => {
    const params: z.infer<typeof queryLogsPayload> = {
      limit,
      startTime: timestamps.startTime,
      endTime: timestamps.endTime,
      host: { filters: [] },
      requestId: { filters: [] },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "startTime":
        case "endTime":
          params[filter.field] = filter.value as number;
          break;

        case "status": {
          params.status?.filters.push({
            operator: "is",
            value: Number.parseInt(filter.value as string),
          });
          break;
        }

        case "methods": {
          if (typeof filter.value !== "string") {
            console.error("Method filter value type has to be 'string'");
            return;
          }
          params.method?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }

        case "paths": {
          if (typeof filter.value !== "string") {
            console.error("Path filter value type has to be 'string'");
            return;
          }
          params.path?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "host": {
          if (typeof filter.value !== "string") {
            console.error("Host filter value type has to be 'string'");
            return;
          }
          params.host?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }

        case "requestId": {
          if (typeof filter.value !== "string") {
            console.error("Request ID filter value type has to be 'string'");
            return;
          }
          params.requestId?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }
      }
    });

    return params;
  }, [filters, limit, timestamps]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.logs.queryLogs.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialCursor: { requestId: null, time: null },
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const result = await queryClient.logs.queryLogs.fetch({
        ...queryParams,
        startTime: [...realtimeLogs, ...historicalLogs][0]?.time ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
      });

      if (result.logs.length > 0) {
        const existingRequestIds = new Set(
          [...realtimeLogs, ...historicalLogs].map((log) => log.request_id),
        );

        const newLogs = result.logs.filter((newLog) => !existingRequestIds.has(newLog.request_id));

        if (newLogs.length > 0) {
          setRealtimeLogs((prev) => {
            const combined = [...newLogs, ...prev];
            // Keep realtime logs limited to avoid memory issues
            return combined.slice(0, Math.min(limit, 100));
          });
        }
      }
    } catch (error) {
      console.error("Error polling for new logs:", error);
    }
  }, [realtimeLogs, historicalLogs, queryParams, queryClient, limit, pollIntervalMs]);

  // Set up polling effect
  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  // Initialize historical logs from initial query
  useEffect(() => {
    if (initialData) {
      const allLogs = initialData.pages.flatMap((page) => page.logs);
      setHistoricalLogs(allLogs);
    }
  }, [initialData]);

  // Reset realtime logs and refetch first page of the historic data when polling is disabled
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogs([]);
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
