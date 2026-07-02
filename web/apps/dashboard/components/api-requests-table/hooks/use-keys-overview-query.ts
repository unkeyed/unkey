import { keysOverviewFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useSort } from "@/components/logs/hooks/use-sort";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  computeTotalPages,
  usePaginatedNavigation,
  usePaginatedPage,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES, type KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import type { KeysQueryOverviewLogsPayload, SortFields } from "../schema/keys-overview.schema";

type UseLogsQueryParams = {
  limit?: number;
  apiId: string;
};

// This overview is time-windowed and uses the multi-column `useSort` surface
// (URL param `sorts`) that its table wires into directly, so it composes the
// shared pagination primitives rather than usePaginatedListQuery. The
// primitives own page state, the deep-link clamp, and prefetch.
export function useKeysOverviewLogsQuery({ apiId, limit = 50 }: UseLogsQueryParams) {
  const { filters } = useFilters();
  const { sorts } = useSort<SortFields>();
  const { queryTime: timestamp } = useQueryTime();

  // Reset to page 1 when filters, sort, or query time change — the current
  // OFFSET is only meaningful relative to the current ordering, so changing
  // any of these invalidates it.
  const filtersKey = useMemo(
    () =>
      `${filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|")}|t:${timestamp}|s:${sorts.map((s) => `${s.column}:${s.direction}`).join(",")}`,
    [filters, timestamp, sorts],
  );

  const { page, setPage } = usePaginatedPage(filtersKey);

  // Check if user explicitly set a time frame filter
  const hasTimeFrameFilter = useMemo(() => {
    return filters.some((filter) => filter.field === "startTime" || filter.field === "endTime");
  }, [filters]);

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
      page,
      useTimeFrameFilter: hasTimeFrameFilter,
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
  }, [filters, limit, timestamp, apiId, sorts, hasTimeFrameFilter, page]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.api.keys.query.useQuery(
    queryParams,
    PAGINATED_LIST_QUERY_OPTIONS,
  );

  const totalCount = data?.total ?? 0;
  const totalPages = computeTotalPages(totalCount, limit);

  const { onPageChange } = usePaginatedNavigation({
    data,
    page,
    totalPages,
    setPage,
    queryParams,
    prefetch: (params) => utils.api.keys.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
  });

  const historicalLogs = useMemo(() => {
    if (!data) {
      return [];
    }
    // Dedupe by key_id — each row in the overview represents a key, and
    // request_id can be "" when a key's latest activity falls in a completed
    // hour (the hourly aggregate table doesn't preserve request_id).
    const map = new Map<string, KeysOverviewLog>();
    data.keysOverviewLogs.forEach((log) => {
      map.set(log.key_id, log);
    });
    return Array.from(map.values());
  }, [data]);

  const isInitialLoading = isLoading && !data;
  const isNavigating = isFetching && !isInitialLoading;

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
