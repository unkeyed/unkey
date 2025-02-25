import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { useEffect, useMemo, useState } from "react";
import type { AuditQueryLogsPayload } from "../query-logs.schema";
import { useFilters } from "../../../hooks/use-filters";

type UseLogsQueryParams = {
  limit?: number;
};

export function useRatelimitOverviewLogsQuery({
  limit = 50,
}: UseLogsQueryParams) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, RatelimitOverviewLog>()
  );

  const { filters } = useFilters();

  const historicalLogs = useMemo(
    () => Array.from(historicalLogsMap.values()),
    [historicalLogsMap]
  );

  //Required for preventing double trpc call during initial render
  const dateNow = useMemo(() => Date.now(), []);
  const queryParams = useMemo(() => {
    const params: AuditQueryLogsPayload = {
      limit,
      startTime: dateNow - HISTORICAL_DATA_WINDOW,
      endTime: dateNow,
      events: { filters: [] },
      users: { filters: [] },
      rootKeys: { filters: [] },
      bucket: "",
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "events": {
          if (typeof filter.value !== "string") {
            console.error("Events filter value type has to be 'string'");
            return;
          }
          params.events?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
        case "rootKeys": {
          if (typeof filter.value !== "string") {
            console.error("RootKeys filter value type has to be 'string'");
            return;
          }
          params.rootKeys?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "users": {
          if (typeof filter.value !== "string") {
            console.error("Users filter value type has to be 'string'");
            return;
          }
          params.users?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(
              `${filter.field} filter value type has to be 'string'`
            );
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
  }, [filters, limit, dateNow]);

  const {
    data: initialData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = trpc.audit.logs.useInfiniteQuery(queryParams, {
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
    isLoading,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
