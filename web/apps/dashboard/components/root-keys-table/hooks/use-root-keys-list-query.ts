import {
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import type { RootKeysFilterValue } from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import { parseAsSortArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { serializeFilters } from "@/hooks/serialize-transition-key";
import { usePageChange } from "@/hooks/use-page-change";
import { usePageClamp } from "@/hooks/use-page-clamp";
import { usePageTransition } from "@/hooks/use-page-transition";
import { usePrefetchPages } from "@/hooks/use-prefetch-pages";
import { trpc } from "@/lib/trpc/client";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useMemo } from "react";
import type { RootKeysQueryPayload, RootKeysSortField } from "../schema/query-logs.schema";

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
    if (!fieldConfig || !fieldConfig.operators.includes(filter.operator)) {
      continue;
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

const MAX_PAGE_SIZE = 200;

export function useRootKeysListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const normalizedPageSize =
    Number.isFinite(pageSize) && pageSize > 0
      ? Math.min(Math.floor(pageSize), MAX_PAGE_SIZE)
      : DEFAULT_PAGE_SIZE;

  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);
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
  const filtersKey = useMemo(() => serializeFilters(filters), [filters]);

  const queryPage = usePageTransition({
    transitionKey: filtersKey,
    page: normalizedPage,
    setPage,
  });

  const baseParams = useMemo<RootKeysFilterParams>(() => buildQueryParams(filters), [filters]);

  const queryParams = useMemo(
    () => ({
      ...baseParams,
      page: queryPage,
      limit: normalizedPageSize,
      sortBy: sortParams?.[0]?.column ?? "createdAt",
      sortOrder: sortParams?.[0]?.direction ?? "desc",
    }),
    [baseParams, queryPage, normalizedPageSize, sortParams],
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
  const totalPages = Math.max(1, Math.ceil(totalCount / normalizedPageSize));

  usePageClamp({
    page: queryPage,
    totalPages,
    data,
    setPage,
  });

  usePrefetchPages({
    page: queryPage,
    totalPages,
    queryParams,
    prefetch: (params) =>
      utils.settings.rootKeys.query.prefetch(params, { staleTime: Number.POSITIVE_INFINITY }),
  });

  const onPageChange = usePageChange(totalPages, setPage);

  return {
    rootKeys: data?.keys ?? [],
    isLoading,
    isInitialLoading,
    isPending: isFetching,
    isFetching,
    page: queryPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
