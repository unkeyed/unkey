import { keyDetailsFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { serializeFilters } from "@/hooks/serialize-transition-key";
import { usePageChange } from "@/hooks/use-page-change";
import { usePageClamp } from "@/hooks/use-page-clamp";
import { usePageTransition } from "@/hooks/use-page-transition";
import { usePrefetchPages } from "@/hooks/use-prefetch-pages";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { parseAsInteger, useQueryState } from "nuqs";
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

  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  // Filters and the time window both invalidate the current page; a
  // transition also clears the realtime buffer since buffered live rows
  // belong to the previous result set.
  const filtersKey = useMemo(
    () => `${serializeFilters(filters)}|ts:${timestamp}`,
    [filters, timestamp],
  );

  const queryPage = usePageTransition({
    transitionKey: filtersKey,
    page: normalizedPage,
    setPage,
    onTransition: () => setRealtimeLogsMap(new Map()),
  });

  const activeRealtimeLogsMap = useMemo(() => {
    return startPolling && queryPage === 1 ? realtimeLogsMap : new Map<string, KeyDetailsLog>();
  }, [startPolling, queryPage, realtimeLogsMap]);

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
      page: queryPage,
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
  }, [filters, limit, timestamp, keyId, keyspaceId, queryPage]);

  // Main query for historical data
  const {
    data: logData,
    isLoading,
    isFetching,
  } = trpc.key.logs.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

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

  const totalCount = useMemo(() => {
    return logData?.total ?? 0;
  }, [logData]);

  const totalPages = Math.max(1, Math.ceil(totalCount / limit));

  usePageClamp({
    page: queryPage,
    totalPages,
    data: logData,
    setPage,
  });

  usePrefetchPages({
    page: queryPage,
    totalPages,
    queryParams,
    prefetch: (params) =>
      queryClient.key.logs.query.prefetch(params, { staleTime: Number.POSITIVE_INFINITY }),
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
    if (startPolling && queryPage === 1) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, queryPage, pollForNewLogs, pollIntervalMs]);

  const onPageChange = usePageChange(totalPages, setPage);

  const isInitialLoading = isLoading && !logData;
  const isNavigating = isFetching && !isInitialLoading;

  return {
    realtimeLogs,
    historicalLogs,
    totalCount: totalCount || 0,
    isLoading: isInitialLoading,
    isFetching,
    isNavigating,
    isPolling: startPolling,
    page: queryPage,
    pageSize: limit,
    totalPages,
    onPageChange,
  };
}

// Helper function to sort logs by time in descending order (newest first)
const sortLogs = (logs: KeyDetailsLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
