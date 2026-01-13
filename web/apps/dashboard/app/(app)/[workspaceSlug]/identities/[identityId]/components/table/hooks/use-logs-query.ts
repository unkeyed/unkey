import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { identityDetailsFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { IdentityLogsPayload } from "../query-logs.schema";

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
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();
  const [realtimeLogs, setRealtimeLogs] = useState<IdentityLog[]>([]);
  const [lastPolledTime, setLastPolledTime] = useState<number | null>(null);

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
  const {
    data,
    isLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = trpc.identity.logs.query.useInfiniteQuery(
    queryParams,
    {
      getNextPageParam: (lastPage: any) => lastPage.nextCursor,
      refetchInterval: false,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    },
  );

  // Polling query for real-time updates
  const pollingParams = useMemo(() => {
    if (!startPolling || !lastPolledTime) return null;

    return {
      ...queryParams,
      startTime: lastPolledTime,
      endTime: Date.now(),
      limit: 100, // Get more recent logs
    };
  }, [queryParams, startPolling, lastPolledTime]);

  const { data: pollingData } = trpc.identity.logs.query.useQuery(
    pollingParams!,
    {
      enabled: !!pollingParams && startPolling,
      refetchInterval: startPolling ? pollIntervalMs : false,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    },
  );

  // Update realtime logs when polling data changes
  useEffect(() => {
    if (pollingData?.logs && pollingData.logs.length > 0) {
      setRealtimeLogs((prev) => {
        const newLogs = pollingData.logs.filter(
          (newLog: IdentityLog) => !prev.some((existingLog) => existingLog.request_id === newLog.request_id),
        );
        return [...newLogs, ...prev].slice(0, 100); // Keep only recent 100 logs
      });
      setLastPolledTime(Date.now());
    }
  }, [pollingData]);

  // Initialize polling timestamp when starting
  useEffect(() => {
    if (startPolling && !lastPolledTime) {
      setLastPolledTime(Date.now());
    } else if (!startPolling) {
      setRealtimeLogs([]);
      setLastPolledTime(null);
    }
  }, [startPolling, lastPolledTime]);

  const historicalLogs = useMemo(() => {
    return data?.pages.flatMap((page: any) => page.logs) ?? [];
  }, [data]);

  const totalCount = useMemo(() => {
    const baseTotal = data?.pages[0]?.total ?? 0;
    // Add realtime logs count to the base total since they represent new logs
    // that weren't included in the original total count
    return baseTotal + realtimeLogs.length;
  }, [data, realtimeLogs.length]);

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
  };
};