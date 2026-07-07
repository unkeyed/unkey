import type {
  RatelimitQueryOverviewLogsPayload,
  SortFields,
} from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useSort } from "@/components/logs/hooks/use-sort";
import { usePageClamp } from "@/hooks/use-page-clamp";
import { usePageTransition } from "@/hooks/use-page-transition";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo } from "react";

type UseRatelimitsOverviewListQueryParams = {
  limit?: number;
  namespaceId: string;
};

const PREFETCH_PAGES_AHEAD = 2;

export const RATELIMITS_OVERVIEW_PAGE_SIZE = 50;

export function useRatelimitsOverviewListPaginated({
  namespaceId,
  limit = RATELIMITS_OVERVIEW_PAGE_SIZE,
}: UseRatelimitsOverviewListQueryParams) {
  const { filters } = useFilters();
  const { sorts } = useSort<SortFields>();
  const { queryTime: timestamp } = useQueryTime();

  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  // Filters, query time, and sort all invalidate the current OFFSET, so any
  // of them changing resets pagination.
  const filtersKey = useMemo(
    () =>
      `${filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|")}|t:${timestamp}|s:${sorts.map((s) => `${s.column}:${s.direction}`).join(",")}`,
    [filters, timestamp, sorts],
  );

  const queryPage = usePageTransition({
    transitionKey: filtersKey,
    page: normalizedPage,
    setPage,
  });

  const queryParams = useMemo<RatelimitQueryOverviewLogsPayload>(() => {
    const params: RatelimitQueryOverviewLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      identifiers: { filters: [] },
      status: { filters: [] },
      namespaceId,
      since: "",
      page: queryPage,
      sorts: sorts.length > 0 ? sorts : null,
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "identifiers": {
          if (typeof filter.value !== "string") {
            return;
          }
          params.identifiers?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "status": {
          if (filter.value !== "blocked" && filter.value !== "passed") {
            return;
          }
          params.status?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }

        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            return;
          }
          params[filter.field] = filter.value;
          break;
        }

        case "since": {
          if (typeof filter.value !== "string") {
            return;
          }
          params.since = filter.value;
          break;
        }
      }
    });

    return params;
  }, [filters, limit, timestamp, namespaceId, sorts, queryPage]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.ratelimit.overview.logs.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const totalCount = Math.max(0, data?.total ?? 0);
  const totalPages = Math.max(1, Math.ceil(totalCount / limit));

  usePageClamp({
    page: queryPage,
    totalPages,
    data,
    setPage,
  });

  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = queryPage + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.ratelimit.overview.logs.query.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [queryPage, totalPages, queryParams, utils.ratelimit.overview.logs.query]);

  const historicalLogs = data?.ratelimitOverviewLogs ?? [];

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );

  const isInitialLoading = isLoading && !data;
  const isNavigating = isFetching && !isInitialLoading;

  return {
    historicalLogs,
    isLoading: isInitialLoading,
    isFetching,
    isNavigating,
    page: queryPage,
    pageSize: limit,
    totalPages,
    totalCount,
    onPageChange,
  };
}
