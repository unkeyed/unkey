import { HISTORICAL_DATA_WINDOW } from "@/app/(app)/logs/constants";
import { trpc } from "@/lib/trpc/client";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { useEffect, useMemo, useState } from "react";
import { useFilters } from "../../../hooks/use-filters";
import type { RatelimitQueryOverviewLogsPayload } from "../query-logs.schema";

type UseLogsQueryParams = {
  limit?: number;
  namespaceId: string;
};

export function useRatelimitOverviewLogsQuery({ namespaceId, limit = 50 }: UseLogsQueryParams) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, RatelimitOverviewLog>(),
  );

  const { filters } = useFilters();

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  //Required for preventing double trpc call during initial render
  const dateNow = useMemo(() => Date.now(), []);
  const queryParams = useMemo(() => {
    const params: RatelimitQueryOverviewLogsPayload = {
      limit,
      startTime: dateNow - HISTORICAL_DATA_WINDOW,
      endTime: dateNow,
      identifiers: { filters: [] },
      status: { filters: [] },
      namespaceId,
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "identifiers": {
          if (typeof filter.value !== "string") {
            console.error("Identifiers filter value type has to be 'string'");
            return;
          }
          params.identifiers?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "status": {
          if (typeof filter.value !== "string") {
            console.error("Status filter value type has to be 'string'");
            return;
          }
          params.status?.filters.push({
            operator: "is",
            value: filter.value as "blocked" | "passed",
          });
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'string'`);
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
  }, [filters, limit, dateNow, namespaceId]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.ratelimit.overview.logs.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialCursor: { requestId: null, time: null },
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Update historical logs effect
  useEffect(() => {
    if (initialData) {
      const newMap = new Map<string, RatelimitOverviewLog>();
      initialData.pages.forEach((page) => {
        page.ratelimitOverviewLogs.forEach((log) => {
          newMap.set(log.identifier, log);
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
