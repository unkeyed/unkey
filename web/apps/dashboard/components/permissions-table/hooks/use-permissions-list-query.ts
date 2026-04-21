import {
  permissionsFilterFieldConfig,
  permissionsListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import type { PermissionsFilterValue } from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/authorization/permissions/hooks/use-filters";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  usePaginatedListQuery,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import type { PermissionsQueryPayload, PermissionsSortField } from "../schema/permissions.schema";

// Mirrors DEFAULT_LIMIT in query.ts
const DEFAULT_PAGE_SIZE = 50;
// Must match the server-side schema max (permissions.schema.ts)
const MAX_PAGE_SIZE = 100;

// Maps TanStack column IDs → server sort field names (and reverse)
const COLUMN_ID_TO_SORT_FIELD: Record<string, PermissionsSortField> = {
  permission: "name",
  slug: "slug",
  used_in_roles: "totalConnectedRoles",
  assigned_to_keys: "totalConnectedKeys",
  last_updated: "lastUpdated",
};
const SORT_FIELD_TO_COLUMN_ID: Record<PermissionsSortField, string> = {
  name: "permission",
  slug: "slug",
  totalConnectedRoles: "used_in_roles",
  totalConnectedKeys: "assigned_to_keys",
  lastUpdated: "last_updated",
};

type PermissionsFilterParams = Pick<
  PermissionsQueryPayload,
  "name" | "description" | "slug" | "roleId" | "roleName"
>;

type PermissionsResponse = { permissions: Permission[]; total: number };

export function usePermissionsListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const result = usePaginatedListQuery<
    PermissionsResponse,
    PermissionsFilterValue,
    PermissionsSortField,
    PermissionsFilterParams
  >({
    pageSize,
    defaultPageSize: DEFAULT_PAGE_SIZE,
    maxPageSize: MAX_PAGE_SIZE,
    defaultSortField: "lastUpdated",
    columnIdToSortField: COLUMN_ID_TO_SORT_FIELD,
    sortFieldToColumnId: SORT_FIELD_TO_COLUMN_ID,
    useFilters,
    filterFieldNames: permissionsListFilterFieldNames,
    filterFieldConfig: permissionsFilterFieldConfig,
    useListQuery: (params) =>
      trpc.authorization.permissions.query.useQuery(params, PAGINATED_LIST_QUERY_OPTIONS),
    usePrefetchNextPage: () => {
      const utils = trpc.useUtils();
      return (params) =>
        utils.authorization.permissions.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS);
    },
  });

  return {
    permissions: result.data?.permissions ?? [],
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
