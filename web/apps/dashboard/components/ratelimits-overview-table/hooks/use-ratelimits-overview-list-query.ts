import { useMemo } from "react";
import type {
  RatelimitQueryOverviewLogsPayload,
  SortFields,
} from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useSort } from "@/components/logs/hooks/use-sort";
import {
  computeTotalPages,
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  paginationFilterKey,
  paginationSortKey,
  usePaginatedNavigation,
  usePaginatedPage,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";

type UseRatelimitsOverviewListQueryParams = {
  limit?: number;
  namespaceId: string;
};

export const RATELIMITS_OVERVIEW_PAGE_SIZE = 50;

// Time-windowed overview using the multi-column `useSort` surface (URL param
// `sorts`). Composes the shared pagination primitives — which own page state,
// the deep-link clamp, and prefetch — while keeping the feature-specific query
// shape here.
export function useRatelimitsOverviewListPaginated({
  namespaceId,
  limit = RATELIMITS_OVERVIEW_PAGE_SIZE,
}: UseRatelimitsOverviewListQueryParams) {
  const { filters } = useFilters();
  const { sorts } = useSort<SortFields>();
  const { queryTime: timestamp } = useQueryTime();

  // Reset to page 1 when filters, sort, or query time change — the current
  // OFFSET is only meaningful relative to the current ordering.
  const filtersKey = useMemo(
    () => `${paginationFilterKey(filters)}|t:${timestamp}|s:${paginationSortKey(sorts)}`,
    [filters, timestamp, sorts],
  );

  const { page, setPage } = usePaginatedPage(filtersKey);

  const queryParams = useMemo<RatelimitQueryOverviewLogsPayload>(() => {
    const params: RatelimitQueryOverviewLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      identifiers: { filters: [] },
      status: { filters: [] },
      namespaceId,
      since: "",
      page,
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
  }, [filters, limit, timestamp, namespaceId, sorts, page]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.ratelimit.overview.logs.query.useQuery(
    queryParams,
    PAGINATED_LIST_QUERY_OPTIONS,
  );

  const totalCount = Math.max(0, data?.total ?? 0);
  const totalPages = computeTotalPages(totalCount, limit);

  const { onPageChange, isInitialLoading, isNavigating } = usePaginatedNavigation({
    data,
    page,
    totalPages,
    setPage,
    isLoading,
    isFetching,
    queryParams,
    prefetch: (params) =>
      utils.ratelimit.overview.logs.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
  });

  const historicalLogs = data?.ratelimitOverviewLogs ?? [];

  return {
    historicalLogs,
    isLoading: isInitialLoading,
    isFetching,
    isNavigating,
    page,
    pageSize: limit,
    totalPages,
    totalCount,
    onPageChange,
  };
}
