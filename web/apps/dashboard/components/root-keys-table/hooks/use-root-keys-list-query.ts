import {
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import type { RootKeysFilterValue } from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import { parseAsSortArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { trpc } from "@/lib/trpc/client";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import type { RootKeysQueryPayload, RootKeysSortField } from "../schema/query-logs.schema";

const PREFETCH_PAGES_AHEAD = 2;

type RootKeysFilterParams = Pick<RootKeysQueryPayload, "name" | "start" | "permission">;

// Mirrors LIMIT in query.ts — kept here to avoid importing the server-side router into the client bundle
const DEFAULT_PAGE_SIZE = 50;

// Maps TanStack column IDs → server sort field names (and reverse)
const COLUMN_ID_TO_SORT_FIELD: Record<string, RootKeysSortField> = {
  root_key: "name",
  created_at: "createdAt",
  last_updated: "lastUpdatedAt",
};
const SORT_FIELD_TO_COLUMN_ID: Record<RootKeysSortField, string> = {
  name: "root_key",
  createdAt: "created_at",
  lastUpdatedAt: "last_updated",
};

function buildQueryParams(filters: RootKeysFilterValue[]): RootKeysFilterParams {
  const params: RootKeysFilterParams = {
    name: [],
    start: [],
    permission: [],
  };

  for (const filter of filters) {
    if (!rootKeysListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
      continue;
    }

    const fieldConfig = rootKeysFilterFieldConfig[filter.field];
    if (!fieldConfig.operators.includes(filter.operator)) {
      throw new Error("Invalid operator");
    }

    if (typeof filter.value === "string") {
      params[filter.field]?.push({
        operator: filter.operator,
        value: filter.value,
      });
    }
  }

  return params;
}

export function useRootKeysListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const [sortParams, setSortParams] = useQueryState("sort", parseAsSortArray<RootKeysSortField>());

  const sorting: SortingState = useMemo(() => {
    if (!sortParams || sortParams.length === 0) {
      return [{ id: "created_at", desc: true }];
    }
    return sortParams.map((s) => ({
      id: SORT_FIELD_TO_COLUMN_ID[s.column] ?? s.column,
      desc: s.direction === "desc",
    }));
  }, [sortParams]);

  const onSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      setSortParams(
        next.length === 0
          ? null
          : next
              .filter((s) => COLUMN_ID_TO_SORT_FIELD[s.id] !== undefined)
              .map((s) => ({
                column: COLUMN_ID_TO_SORT_FIELD[s.id],
                direction: s.desc ? "desc" : "asc",
              })),
      );
      setPage(1);
    },
    [sorting, setSortParams, setPage],
  );

  // Stable string key derived from filter content — avoids resetting page when
  // useQueryStates returns a new array reference for the same filter values
  // (which happens on every URL change, including page navigation).
  const filtersKey = useMemo(
    () => filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|"),
    [filters],
  );

  // Reset to page 1 only when filter content actually changes (not on initial mount).
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

  const baseParams = useMemo<RootKeysFilterParams>(() => buildQueryParams(filters), [filters]);

  const queryParams = useMemo(
    () => ({
      ...baseParams,
      page,
      limit: pageSize,
      sortBy: sortParams?.[0]?.column ?? "createdAt",
      sortOrder: sortParams?.[0]?.direction ?? "desc",
    }),
    [baseParams, page, pageSize, sortParams],
  );

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.settings.rootKeys.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const isInitialLoading = isLoading && !data;

  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));

  // Clamp page to valid range after data/totalPages updates.
  useEffect(() => {
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [page, totalPages, setPage]);

  // Prefetch the next few pages so navigation feels instant.
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = page + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.settings.rootKeys.query.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [page, totalPages, queryParams, utils.settings.rootKeys.query]);

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );

  return {
    rootKeys: data?.keys ?? [],
    isLoading,
    isInitialLoading,
    isPending: isFetching,
    isFetching,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
