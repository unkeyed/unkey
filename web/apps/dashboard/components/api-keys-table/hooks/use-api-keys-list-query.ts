import {
  keysListFilterFieldConfig,
  type keysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/hooks/use-filters";
import { parseAsSortArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { PAGINATED_LIST_QUERY_OPTIONS, normalizePageSize } from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useState } from "react";
import type {
  ApiKeysCursor,
  ApiKeysQueryPayload,
  ApiKeysSortField,
} from "../schema/api-keys.schema";

const DEFAULT_PAGE_SIZE = 50;

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

type ApiKeysFilterParams = Pick<ApiKeysQueryPayload, (typeof keysListFilterFieldNames)[number]>;

type UseApiKeysListQueryParams = {
  keyAuthId: string;
  pageSize?: number;
};

export function useApiKeysListQuery({
  keyAuthId,
  pageSize = DEFAULT_PAGE_SIZE,
}: UseApiKeysListQueryParams) {
  const { filters } = useFilters();
  const [sortParams, setSortParams] = useQueryState("sort", parseAsSortArray<ApiKeysSortField>());
  const [, setLegacyPage] = useQueryState("page", parseAsInteger);
  useEffect(() => {
    setLegacyPage(null);
  }, [setLegacyPage]);

  const sort = sortParams?.find((value) =>
    Object.prototype.hasOwnProperty.call(SORT_FIELD_TO_COLUMN_ID, value.column),
  ) ?? { column: "lastUsedAt", direction: "desc" };
  const sorting: SortingState = [
    { id: SORT_FIELD_TO_COLUMN_ID[sort.column], desc: sort.direction === "desc" },
  ];
  const limit = normalizePageSize(pageSize, DEFAULT_PAGE_SIZE, DEFAULT_PAGE_SIZE);
  const filterParams = useMemo(() => {
    const params: ApiKeysFilterParams = { names: [], identities: [], keyIds: [], tags: [] };
    for (const filter of filters) {
      if (
        typeof filter.value === "string" &&
        keysListFilterFieldConfig[filter.field].operators.includes(filter.operator)
      ) {
        params[filter.field]?.push({ operator: filter.operator, value: filter.value });
      }
    }
    return params;
  }, [filters]);
  const queryParams = {
    ...filterParams,
    keyAuthId,
    limit,
    sortBy: sort.column,
    sortOrder: sort.direction,
  };
  const queryKey = JSON.stringify(queryParams);
  const [navigation, setNavigation] = useState<{
    queryKey: string;
    cursors: (ApiKeysCursor | null)[];
  }>({ queryKey, cursors: [null] });
  const cursors = navigation.queryKey === queryKey ? navigation.cursors : [null];
  if (navigation.queryKey !== queryKey) {
    setNavigation({ queryKey, cursors });
  }

  const { data, isLoading, isFetching, isPreviousData, isError } = trpc.api.keys.list.useQuery(
    { ...queryParams, cursor: cursors.at(-1) },
    PAGINATED_LIST_QUERY_OPTIONS,
  );
  const page = cursors.length;
  const hasNextPage = !isPreviousData && !isError && data?.nextCursor != null;
  const onPageChange = (newPage: number) => {
    if (isFetching) {
      return;
    }
    if (newPage === page - 1 && page > 1) {
      setNavigation({ queryKey, cursors: cursors.slice(0, -1) });
    } else if (newPage === page + 1 && hasNextPage && data?.nextCursor) {
      setNavigation({ queryKey, cursors: [...cursors, data.nextCursor] });
    }
  };

  const onSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const current: SortingState = [
        { id: SORT_FIELD_TO_COLUMN_ID[sort.column], desc: sort.direction === "desc" },
      ];
      const next = typeof updater === "function" ? updater(current) : updater;
      const first = next.find((value) =>
        Object.prototype.hasOwnProperty.call(COLUMN_ID_TO_SORT_FIELD, value.id),
      );
      setSortParams(
        first
          ? [{ column: COLUMN_ID_TO_SORT_FIELD[first.id], direction: first.desc ? "desc" : "asc" }]
          : null,
      );
    },
    [sort.column, sort.direction, setSortParams],
  );

  const isInitialLoading = isLoading && !data;
  return {
    keys: data?.keys ?? [],
    isLoading: isInitialLoading,
    isInitialLoading,
    isNavigating: isFetching && !isInitialLoading,
    page,
    hasNextPage,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
