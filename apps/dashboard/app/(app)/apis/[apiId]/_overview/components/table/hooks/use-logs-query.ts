import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useSort } from "@/components/logs/hooks/use-sort";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES, type KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useEffect, useMemo, useState } from "react";
import { keysOverviewFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { KeysQueryOverviewLogsPayload, SortFields } from "../query-logs.schema";

type UseLogsQueryParams = {
  limit?: number;
  apiId: string;
};

export function useKeysOverviewLogsQuery({ apiId, limit = 50 }: UseLogsQueryParams) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, KeysOverviewLog>(),
  );

  const { filters } = useFilters();
  const { sorts } = useSort<SortFields>();

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: KeysQueryOverviewLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      keyIds: [],
      outcomes: [],
      identities: [],
      names: [],
      tags: [],
      apiId,
      since: "",
      sorts: sorts.length > 0 ? sorts : null,
    };

    filters.forEach((filter) => {
      const fieldConfig = keysOverviewFilterFieldConfig[filter.field];
      const validOperators = fieldConfig.operators;

      const operator = validOperators.includes(filter.operator)
        ? filter.operator
        : validOperators[0];

      switch (filter.field) {
        case "keyIds": {
          if (typeof filter.value === "string") {
            const keyIdOperator = operator === "is" || operator === "contains" ? operator : "is";

            params.keyIds?.push({
              operator: keyIdOperator,
              value: filter.value,
            });
          }
          break;
        }

        case "names":
        case "identities":
          if (typeof filter.value === "string") {
            params[filter.field]?.push({
              operator,
              value: filter.value,
            });
          }
          break;

        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes?.push({
              operator: "is",
              value: filter.value as ValidOutcome,
            });
          }
          break;
        }

        case "tags": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            params.tags?.push({
              operator,
              value: filter.value,
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
  }, [filters, limit, timestamp, apiId, sorts]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.api.keys.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Update historical logs effect
  useEffect(() => {
    if (initialData) {
      const newMap = new Map<string, KeysOverviewLog>();
      initialData.pages.forEach((page) => {
        page.keysOverviewLogs.forEach((log) => {
          // Use request_id as the unique key since key_id might not be unique across different requests
          newMap.set(log.request_id, log);
        });
      });
      setHistoricalLogsMap(newMap);
    }
  }, [initialData]);

  return {
    historicalLogs,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
