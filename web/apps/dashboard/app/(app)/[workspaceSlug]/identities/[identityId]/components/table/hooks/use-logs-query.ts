import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useCallback, useEffect, useMemo, useState } from "react";
import { identityDetailsFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { IdentityLogsPayload } from "../query-logs.schema";

// Maximum number of real-time logs to store
const REALTIME_DATA_LIMIT = 100;

type UseIdentityLogsQueryProps = {
  identityId: string;
  startPolling?: boolean;
  pollIntervalMs?: number;
};

export const useIdentityLogsQuery = ({
  identityId,
  startPolling = false,
  pollIntervalMs = 2000,
}: UseIdentityLogsQueryProps) => {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(() => new Map<string, IdentityLog>());
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, IdentityLog>());
  const [totalCount, setTotalCount] = useState(0);

  const { filters } = useFilters();
  const queryClient = trpc.useUtils();
  const { queryTime: timestamp } = useQueryTime();

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(realtimeLogsMap.values()));
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const queryParams = useMemo(() => {
    const params: IdentityLogsPayload = {
      identityId,
      limit: 50,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      since: "",
      cursor: null,
      tags: null,
      outcomes: null,
    };

    filters.forEach((filter) => {
      if (!(filter.field in identityDetailsFilterFieldConfig)) {
        return;
      }

      switch (filter.field) {
        case "tags": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            const fieldConfig = identityDetailsFilterFieldConfig[filter.field];
            const validOperators = fieldConfig.operators;

            const operator = validOperators.includes(filter.operator)
              ? filter.operator
              : validOperators[0];

            params.tags = [
              {
                operator,
                value: filter.value,
              },
            ];
          }
          break;
        }

        case "startTime":
        case "endTime": {
          // TypeScript knows filter.value is number for these fields
          params[filter.field] = filter.value;
          break;
        }

        case "since": {
          // TypeScript knows filter.value is string for this field
          params.since = filter.value;
          break;
        }

        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            if (!params.outcomes) {
              params.outcomes = [];
            }
            params.outcomes.push({
              operator: "is",
              value: filter.value as ValidOutcome,
            });
          }
          break;
        }
      }
    });

    return params;
  }, [filters, timestamp, identityId]);

  // Main query for historical data
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    trpc.identity.logs.query.useInfiniteQuery(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      refetchInterval: false,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    });

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;

      const result = await queryClient.identity.logs.query.fetch({
        ...queryParams,
        startTime: latestTime ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
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
          if (newMap.size > Math.min(50, REALTIME_DATA_LIMIT)) {
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
      console.error("Error polling for new identity logs:", error);
    }
  }, [queryParams, queryClient, pollIntervalMs, historicalLogsMap, realtimeLogs, historicalLogs]);

  // Set up polling effect
  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  // Update historical logs effect
  useEffect(() => {
    if (data) {
      const newMap = new Map<string, IdentityLog>();
      data.pages.forEach((page) => {
        page.logs.forEach((log) => {
          newMap.set(log.request_id, log);
        });
      });
      setHistoricalLogsMap(newMap);

      if (data.pages.length > 0) {
        setTotalCount(data.pages[0].total);
      }
    }
  }, [data]);

  // Reset realtime logs effect
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  const loadMore = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return {
    realtimeLogs,
    historicalLogs,
    isLoading,
    isLoadingMore: isFetchingNextPage,
    loadMore,
    hasMore: hasNextPage,
    totalCount,
    isPolling: startPolling,
  };
};

// Helper function to sort logs by time in descending order (newest first)
const sortLogs = (logs: IdentityLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
