import { keyDetailsFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useLoadMoreList } from "@/hooks/use-load-more-list";
import { paginationFilterKey } from "@/hooks/use-paginated-list-query";
import { getTimestampFromRelative } from "@/lib/duration";
import { trpcClient } from "@/lib/trpc/client";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { KeyDetailsLogsPayload } from "../schema/query-logs.schema";

// Maximum number of real-time logs to store
const REALTIME_DATA_LIMIT = 100;

type KeyDetailsLogsPage = inferRouterOutputs<Router>["key"]["logs"]["query"];

const rowsOf = (response: KeyDetailsLogsPage) => response.logs;
const totalOf = (response: KeyDetailsLogsPage) => response.total;
const requestIdOf = (log: KeyDetailsLog) => log.request_id;

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

  const filtersKey = useMemo(
    () => `${paginationFilterKey(filters)}|ts:${timestamp}`,
    [filters, timestamp],
  );

  const [prevFiltersKey, setPrevFiltersKey] = useState(filtersKey);
  if (prevFiltersKey !== filtersKey) {
    setPrevFiltersKey(filtersKey);
    setRealtimeLogsMap(new Map());
  }

  const activeRealtimeLogsMap = useMemo(() => {
    return startPolling ? realtimeLogsMap : new Map<string, KeyDetailsLog>();
  }, [startPolling, realtimeLogsMap]);

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(activeRealtimeLogsMap.values()));
  }, [activeRealtimeLogsMap]);

  const queryParams = useMemo(() => {
    const params: Omit<KeyDetailsLogsPayload, "page"> = {
      limit,
      keyId,
      keyspaceId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      outcomes: [],
      tags: [],
      since: "",
    };
    let since = "";

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
            since = filter.value;
          }
          break;
      }
    });

    if (since) {
      params.startTime = getTimestampFromRelative(since, timestamp);
      params.endTime = timestamp;
    }

    return params;
  }, [filters, limit, timestamp, keyId, keyspaceId]);

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.key.logs.query, queryParams, "infinite"),
    fetchPage: (page, signal) =>
      trpcClient.key.logs.query.query({ ...queryParams, page }, { signal }),
    rowsOf,
    idOf: requestIdOf,
    totalOf,
    pageSize: limit,
    paused: startPolling,
  });
  const historicalLogs = list.table.rows;
  const historicalIds = useMemo(() => new Set(historicalLogs.map(requestIdOf)), [historicalLogs]);

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
          if (newMap.has(log.request_id) || historicalIds.has(log.request_id)) {
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
    historicalIds,
    realtimeLogs,
    historicalLogs,
  ]);

  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  return {
    realtimeLogs,
    table: list.table,
    totalCount: list.totalCount,
  };
}

// Helper function to sort logs by time in descending order (newest first)
const sortLogs = (logs: KeyDetailsLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
