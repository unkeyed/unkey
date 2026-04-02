import { keysOverviewFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useSort } from "@/components/logs/hooks/use-sort";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES, type KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import type { KeysQueryOverviewLogsPayload, SortFields } from "../schema/keys-overview.schema";

type UseLogsQueryParams = {
  limit?: number;
  apiId: string;
};

const PREFETCH_PAGES_AHEAD = 2;

export function useKeysOverviewLogsQuery({ apiId, limit = 50 }: UseLogsQueryParams) {
  const { filters } = useFilters();
  const { sorts } = useSort<SortFields>();
  const { queryTime: timestamp } = useQueryTime();

  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

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
      page: normalizedPage,
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
  }, [filters, limit, timestamp, apiId, sorts, hasTimeFrameFilter, normalizedPage]);

  // Reset to page 1 when filters or query time change
  const filtersKey = useMemo(
    () => `${filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|")}|t:${timestamp}`,
    [filters, timestamp],
  );

  const prevFiltersKeyRef = useRef<string | null>(null);
  useEffect(() => {
    if (prevFiltersKeyRef.current === null) {
      prevFiltersKeyRef.current = filtersKey;
      return;
    }
    if (filtersKey !== prevFiltersKeyRef.current) {
      prevFiltersKeyRef.current = filtersKey;
      setPage(1);
    }
  }, [filtersKey, setPage]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.api.keys.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / limit));

  // Clamp page to valid range after data/totalPages updates
  useEffect(() => {
    if (normalizedPage > totalPages) {
      setPage(totalPages);
    }
  }, [normalizedPage, totalPages, setPage]);

  // Prefetch the next few pages
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = normalizedPage + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.api.keys.query.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [normalizedPage, totalPages, queryParams, utils.api.keys.query]);

  const historicalLogs = useMemo(() => {
    if (!data) {
      return [];
    }
    const map = new Map<string, KeysOverviewLog>();
    data.keysOverviewLogs.forEach((log) => {
      map.set(log.request_id, log);
    });
    return Array.from(map.values());
  }, [data]);

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
    page: normalizedPage,
    pageSize: limit,
    totalPages,
    totalCount,
    onPageChange,
  };
}
