import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { KEY_VERIFICATION_OUTCOMES, type KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useEffect, useMemo, useState } from "react";
import { useFilters } from "../../../hooks/use-filters";
import type { KeysQueryOverviewLogsPayload } from "../query-logs.schema";

type UseLogsQueryParams = {
  limit?: number;
  apiId: string;
};

export function useKeysOverviewLogsQuery({ apiId, limit = 50 }: UseLogsQueryParams) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, KeysOverviewLog>(),
  );

  const { filters } = useFilters();
  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  //Required for preventing double trpc call during initial render
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = useMemo(() => {
    const params: KeysQueryOverviewLogsPayload = {
      limit,
      startTime: dateNow - HISTORICAL_DATA_WINDOW,
      endTime: dateNow,
      keyIds: [],
      outcomes: [],
      names: [],
      apiId,
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "keyIds": {
          if (typeof filter.value !== "string") {
            console.error("Keys filter value type has to be 'string'");
            return;
          }
          params.keyIds?.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
        case "names": {
          if (typeof filter.value !== "string") {
            console.error("Names filter value type has to be 'string'");
            return;
          }
          params.names?.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
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
          } else {
            console.error("Invalid outcome value:", filter.value);
          }
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'number'`);
            return;
          }
          params[filter.field] = filter.value;
          break;
        }
        case "since": {
          if (typeof filter.value !== "string") {
            console.error("Since filter value type has to be 'string'");
            return;
          }
          params.since = filter.value;
          break;
        }
      }
    });

    return params;
  }, [filters, limit, dateNow, apiId]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.api.keys.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialCursor: { requestId: null, time: null },
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
