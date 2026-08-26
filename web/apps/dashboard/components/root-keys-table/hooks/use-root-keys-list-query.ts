import {
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import type { RootKeysFilterValue } from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  usePaginatedListQuery,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import type { RootKeysQueryPayload, RootKeysSortField } from "../schema/query-logs.schema";

// Mirrors LIMIT in query.ts — kept here to avoid importing the server-side router into the client bundle
const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;

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

type RootKeysFilterParams = Pick<RootKeysQueryPayload, "name">;

type RootKeysResponse = { keys: RootKey[]; total: number };

export function useRootKeysListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const utils = trpc.useUtils();

  const result = usePaginatedListQuery<
    RootKeysResponse,
    RootKeysFilterValue,
    RootKeysSortField,
    RootKeysFilterParams
  >({
    pageSize,
    defaultPageSize: DEFAULT_PAGE_SIZE,
    maxPageSize: MAX_PAGE_SIZE,
    defaultSortField: "createdAt",
    columnIdToSortField: COLUMN_ID_TO_SORT_FIELD,
    sortFieldToColumnId: SORT_FIELD_TO_COLUMN_ID,
    useFilters,
    filterFieldNames: rootKeysListFilterFieldNames,
    filterFieldConfig: rootKeysFilterFieldConfig,
    useListQuery: (params) =>
      // biome-ignore lint/correctness/useHookAtTopLevel: hook factory invoked unconditionally inside the paginated-list hook
      trpc.settings.rootKeys.query.useQuery(params, PAGINATED_LIST_QUERY_OPTIONS),
    prefetch: (params) =>
      utils.settings.rootKeys.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
    getTotalCount: (data) => data.total,
    // Preserve the bespoke hook's behavior: keep the URL clean until the user sorts.
    syncDefaultSortToUrl: false,
  });

  return {
    rootKeys: result.data?.keys ?? [],
    isLoading: result.isInitialLoading,
    isInitialLoading: result.isInitialLoading,
    isNavigating: result.isNavigating,
    page: result.page,
    pageSize: result.pageSize,
    totalPages: result.totalPages,
    totalCount: result.totalCount,
    onPageChange: result.onPageChange,
    sorting: result.sorting,
    onSortingChange: result.onSortingChange,
  };
}
