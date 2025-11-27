import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { useEffect, useMemo, useState } from "react";
import { useSort } from "../../../../../../../../../components/logs/hooks/use-sort";
import { useFilters } from "../../../hooks/use-filters";
import type { RatelimitQueryOverviewLogsPayload, SortFields } from "../query-logs.schema";

import { useInfiniteQuery } from "@tanstack/react-query";

type UseLogsQueryParams = {
  limit?: number;
  namespaceId: string;
};

export function useRatelimitOverviewLogsQuery({ namespaceId, limit = 50 }: UseLogsQueryParams) {
  const trpc = useTRPC();
  const [totalCount, setTotalCount] = useState(0);
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, RatelimitOverviewLog>(),
  );

  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const { sorts } = useSort<SortFields>();

  //Required for preventing double trpc call during initial render

  const queryParams = useMemo(() => {
    const params: RatelimitQueryOverviewLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      identifiers: { filters: [] },
      status: { filters: [] },
      namespaceId,
      since: "",
      sorts: sorts.length > 0 ? sorts : null,
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
  }, [filters, limit, timestamp, namespaceId, sorts]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = useInfiniteQuery(
    trpc.ratelimit.overview.logs.query.infiniteQueryOptions(queryParams, {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }),
  );

  // Update historical logs effect
  useEffect(() => {
    if (initialData) {
      const newMap = new Map<string, RatelimitOverviewLog>();
      initialData.pages.forEach((page) => {
        page.ratelimitOverviewLogs.forEach((log) => {
          newMap.set(log.identifier, log);
        });
      });
      if (initialData.pages.length > 0) {
        setTotalCount(initialData.pages[0].total);
      }
      setHistoricalLogsMap(newMap);
    }
  }, [initialData]);

  return {
    historicalLogs,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    totalCount,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
