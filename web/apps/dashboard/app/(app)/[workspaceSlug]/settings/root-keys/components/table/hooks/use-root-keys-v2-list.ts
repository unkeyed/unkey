import {
  type RootKeysFilterValue,
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import type {
  RootKeysQueryPayload,
  RootKeysSortField,
} from "@/components/root-keys-table/schema/query-logs.schema";
import {
  PAGINATED_LIST_QUERY_OPTIONS,
  usePaginatedListQuery,
} from "@/hooks/use-paginated-list-query";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { listRootKeys, rootKeysV2QueryKeys } from "@/lib/root-keys-api";
import { useQuery } from "@tanstack/react-query";
import { selectRootKeysPage } from "../root-keys-v2";

const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;

const SORT_FIELDS = {
  name: "name",
  createdAt: "createdAt",
  lastUpdatedAt: "lastUpdatedAt",
} as const satisfies Record<RootKeysSortField, RootKeysSortField>;

type RootKeysFilterParams = Pick<RootKeysQueryPayload, "name">;

export function useRootKeysV2List(pageSize = DEFAULT_PAGE_SIZE) {
  const workspace = useWorkspaceNavigation();
  const result = usePaginatedListQuery<
    ReturnType<typeof selectRootKeysPage>,
    RootKeysFilterValue,
    RootKeysSortField,
    RootKeysFilterParams
  >({
    pageSize,
    defaultPageSize: DEFAULT_PAGE_SIZE,
    maxPageSize: MAX_PAGE_SIZE,
    defaultSortField: "createdAt",
    columnIdToSortField: SORT_FIELDS,
    sortFieldToColumnId: SORT_FIELDS,
    useFilters,
    filterFieldNames: rootKeysListFilterFieldNames,
    filterFieldConfig: rootKeysFilterFieldConfig,
    useListQuery: (params) => {
      // biome-ignore lint/correctness/useHookAtTopLevel: hook factory invoked unconditionally inside the paginated-list hook
      const query = useQuery({
        queryKey: rootKeysV2QueryKeys.workspace(workspace.id),
        queryFn: ({ signal }) => listRootKeys(signal),
        ...PAGINATED_LIST_QUERY_OPTIONS,
      });
      return {
        data: query.data ? selectRootKeysPage(query.data, params) : undefined,
        isLoading: query.isLoading,
        isFetching: query.isFetching,
      };
    },
    prefetch: () => undefined,
    getTotalCount: (data) => data.total,
    syncDefaultSortToUrl: false,
  });

  return {
    rootKeys: result.data?.keys ?? [],
    isInitialLoading: result.isInitialLoading,
    isNavigating: result.isNavigating,
    page: result.page,
    pageSize: result.pageSize,
    totalPages: result.totalPages,
    totalCount: result.totalCount,
    onPageChange: result.onPageChange,
  };
}
