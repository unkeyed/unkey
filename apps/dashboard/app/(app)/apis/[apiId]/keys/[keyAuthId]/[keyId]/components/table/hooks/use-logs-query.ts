import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useEffect, useMemo, useState } from "react";
import { keyDetailsFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { KeyDetailsLogsPayload } from "../query-logs.schema";

type UseKeyDetailsLogsQueryParams = {
  limit?: number;
  keyId: string;
  apiId: string;
  keyspaceId: string;
};

export function useKeyDetailsLogsQuery({
  keyId,
  keyspaceId,
  apiId,
  limit = 50,
}: UseKeyDetailsLogsQueryParams) {
  const [logsMap, setLogsMap] = useState(() => new Map<string, KeyDetailsLog>());

  const { filters } = useFilters();

  const logs = useMemo(() => Array.from(logsMap.values()), [logsMap]);

  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: KeyDetailsLogsPayload = {
      limit,
      apiId,
      keyId,
      keyspaceId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      outcomes: [],
      since: "",
    };

    filters.forEach((filter) => {
      const fieldConfig = keyDetailsFilterFieldConfig[filter.field];
      const validOperators = fieldConfig?.operators;

      if (!validOperators) {
        return;
      }

      switch (filter.field) {
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
  }, [filters, limit, timestamp, keyId, keyspaceId, apiId]);

  const {
    data: logData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading,
  } = trpc.key.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Update logs map effect
  useEffect(() => {
    if (logData) {
      const newMap = new Map<string, KeyDetailsLog>();
      logData.pages.forEach((page) => {
        page.logs.forEach((log) => {
          // Use request_id as the unique key
          newMap.set(log.request_id, log);
        });
      });
      setLogsMap(newMap);
    }
  }, [logData]);

  return {
    logs,
    totalCount: logData?.pages[0]?.total || 0,
    isLoading,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
