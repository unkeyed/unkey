import { keyDetailsFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  computeTotalPages,
  paginationFilterKey,
  usePaginatedNavigation,
  usePaginatedPage,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { KeyDetailsLogsPayload } from "../schema/query-logs.schema";

// Maximum number of real-time logs to store
const REALTIME_DATA_LIMIT = 100;

type UseKeyDetailsLogsQueryParams = {
  limit?: number;
  keyId: string;
  keyspaceId: string;
  pollIntervalMs?: number;
  startPolling?: boolean;
};

// Key-details logs are time-windowed, unsorted, and layer live polling on top
// of an offset-paginated history. The shared pagination primitives own page
// state, the deep-link clamp, and prefetch; the realtime buffer and polling
// stay here.
export function useKeyDetailsLogsQuery({
  keyId,
  keyspaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseKeyDetailsLogsQueryParams) {
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, KeyDetailsLog>());

  const { filters } = useFilters();
  const queryClient = trpc.useUtils();
  const { queryTime: timestamp } = useQueryTime();

  // usePaginatedPage owns the page reset off this key; the realtime buffer is
  // cleared here.
  const filtersKey = useMemo(
    () => `${paginationFilterKey(filters)}|ts:${timestamp}`,
    [filters, timestamp],
  );

  const { page, setPage } = usePaginatedPage(filtersKey);

  // Clear the buffer in-render on a filters/time transition, not in an effect:
  // the page stays 1 and `activeRealtimeLogsMap` below is gated on page, so an
  // effect would paint one frame of the old filters' rows against the new ones.
  // State, not a ref, so a discarded render rolls the paired clear back with it.
  const [prevFiltersKey, setPrevFiltersKey] = useState(filtersKey);
  if (prevFiltersKey !== filtersKey) {
    setPrevFiltersKey(filtersKey);
    setRealtimeLogsMap(new Map());
  }

  const activeRealtimeLogsMap = useMemo(() => {
    return startPolling && page === 1 ? realtimeLogsMap : new Map<string, KeyDetailsLog>();
  }, [startPolling, page, realtimeLogsMap]);

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(activeRealtimeLogsMap.values()));
  }, [activeRealtimeLogsMap]);

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
      page,
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
  }, [filters, limit, timestamp, keyId, keyspaceId, page]);

  // Main query for historical data
  const {
    data: logData,
    isLoading,
    isFetching,
  } = trpc.key.logs.query.useQuery(queryParams, PAGINATED_LIST_QUERY_OPTIONS);

  // Derive historical logs from query data
  const historicalLogsMap = useMemo(() => {
    const map = new Map<string, KeyDetailsLog>();
    if (logData) {
      logData.logs.forEach((log) => {
        map.set(log.request_id, log);
      });
    }
    return map;
  }, [logData]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const totalCount = logData?.total ?? 0;
  const totalPages = computeTotalPages(totalCount, limit);

  const { onPageChange, isInitialLoading, isNavigating } = usePaginatedNavigation({
    data: logData,
    page,
    totalPages,
    setPage,
    isLoading,
    isFetching,
    queryParams,
    prefetch: (params) =>
      queryClient.key.logs.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
  });

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;

      const result = await queryClient.key.logs.query.fetch({
        ...queryParams,
        startTime: latestTime ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
        page: 1,
      });

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

  // Set up polling effect — only poll on page 1
  useEffect(() => {
    if (startPolling && page === 1) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, page, pollForNewLogs, pollIntervalMs]);

  return {
    realtimeLogs,
    historicalLogs,
    totalCount: totalCount || 0,
    isLoading: isInitialLoading,
    isFetching,
    isNavigating,
    isPolling: startPolling,
    page,
    pageSize: limit,
    totalPages,
    onPageChange,
  };
}

// Helper function to sort logs by time in descending order (newest first)
const sortLogs = (logs: KeyDetailsLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
