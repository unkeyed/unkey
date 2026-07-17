import {
  keysListFilterFieldConfig,
  keysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import type { KeysListFilterValue } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/hooks/use-filters";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  usePaginatedListQuery,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
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

type ApiKeysFilterParams = Pick<ApiKeysQueryPayload, (typeof keysListFilterFieldNames)[number]>;

type ApiKeysResponse = { keys: KeyDetails[]; totalCount: number };

type UseApiKeysListQueryParams = {
  keyAuthId: string;
  pageSize?: number;
};

export function useApiKeysListQuery({
  keyAuthId,
  pageSize = DEFAULT_PAGE_SIZE,
}: UseApiKeysListQueryParams) {
  const utils = trpc.useUtils();

  const result = usePaginatedListQuery<
    ApiKeysResponse,
    KeysListFilterValue,
    ApiKeysSortField,
    ApiKeysFilterParams
  >({
    pageSize,
    defaultPageSize: DEFAULT_PAGE_SIZE,
    maxPageSize: MAX_PAGE_SIZE,
    defaultSortField: "lastUsedAt",
    columnIdToSortField: COLUMN_ID_TO_SORT_FIELD,
    sortFieldToColumnId: SORT_FIELD_TO_COLUMN_ID,
    useFilters,
    filterFieldNames: keysListFilterFieldNames,
    filterFieldConfig: keysListFilterFieldConfig,
    useListQuery: (params) =>
      // biome-ignore lint/correctness/useHookAtTopLevel: hook factory invoked unconditionally inside the paginated-list hook
      trpc.api.keys.list.useQuery({ ...params, keyAuthId }, PAGINATED_LIST_QUERY_OPTIONS),
    prefetch: (params) =>
      utils.api.keys.list.prefetch({ ...params, keyAuthId }, PAGINATED_LIST_PREFETCH_OPTIONS),
    // keyAuthId rides in the closures above rather than in the query params, so
    // the prefetch effect needs it spelled out to re-warm on a keyspace switch.
    prefetchKey: keyAuthId,
    getTotalCount: (data) => data.totalCount,
    // Preserve the bespoke hook's behavior: keep the URL clean until the user sorts.
    syncDefaultSortToUrl: false,
  });

  return {
    keys: result.data?.keys ?? [],
    isLoading: result.isInitialLoading,
    isInitialLoading: result.isInitialLoading,
    isFetching: result.isFetching,
    page: result.page,
    pageSize: result.pageSize,
    totalPages: result.totalPages,
    totalCount: result.totalCount,
    onPageChange: result.onPageChange,
    sorting: result.sorting,
    onSortingChange: result.onSortingChange,
  };
}
