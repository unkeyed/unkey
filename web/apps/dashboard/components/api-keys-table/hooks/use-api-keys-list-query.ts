import {
  keysListFilterFieldConfig,
  keysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/hooks/use-filters";
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
import type { ApiKeysQueryPayload, ApiKeysSortField } from "../schema/api-keys.schema";

const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;

// Maps TanStack column IDs to server sort field names (and reverse)
const COLUMN_ID_TO_SORT_FIELD: Record<string, ApiKeysSortField> = {
  key: "id",
  value: "start",
  last_used: "lastUsedAt",
};
const SORT_FIELD_TO_COLUMN_ID: Record<ApiKeysSortField, string> = {
  id: "key",
  start: "value",
  lastUsedAt: "last_used",
};

type UseApiKeysListQueryParams = {
  keyAuthId: string;
  pageSize?: number;
};

export function useApiKeysListQuery({
  keyAuthId,
  pageSize: pageSizeProp = DEFAULT_PAGE_SIZE,
}: UseApiKeysListQueryParams) {
  const normalizedPageSize =
    Number.isFinite(pageSizeProp) && pageSizeProp > 0
      ? Math.min(Math.floor(pageSizeProp), MAX_PAGE_SIZE)
      : DEFAULT_PAGE_SIZE;

  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);
  const [sortParams, setSortParams] = useQueryState("sort", parseAsSortArray<ApiKeysSortField>());

  const sorting: SortingState = useMemo(() => {
    if (!sortParams || sortParams.length === 0) {
      return [{ id: "last_used", desc: true }];
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

  const filtersKey = useMemo(() => serializeFilters(filters), [filters]);

  const queryPage = usePageTransition({
    transitionKey: filtersKey,
    page: normalizedPage,
    setPage,
  });

  const queryParams = useMemo(() => {
    const params: ApiKeysQueryPayload = {
      limit: normalizedPageSize,
      page: queryPage,
      ...Object.fromEntries(keysListFilterFieldNames.map((field) => [field, []])),
      keyAuthId,
      sortBy: sortParams?.[0]?.column ?? "lastUsedAt",
      sortOrder: sortParams?.[0]?.direction ?? "desc",
    };

    for (const filter of filters) {
      if (!keysListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        continue;
      }

      const fieldConfig = keysListFilterFieldConfig[filter.field];
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
  }, [filters, keyAuthId, queryPage, normalizedPageSize, sortParams]);

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.api.keys.list.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const isInitialLoading = isLoading && !data;
  const totalCount = data?.totalCount ?? 0;
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
      utils.api.keys.list.prefetch(params, { staleTime: Number.POSITIVE_INFINITY }),
  });

  const onPageChange = usePageChange(totalPages, setPage);

  return {
    keys: data?.keys ?? [],
    isLoading,
    isInitialLoading,
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
