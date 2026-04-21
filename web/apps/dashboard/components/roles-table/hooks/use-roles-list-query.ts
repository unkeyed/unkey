import {
  type RolesFilterValue,
  rolesFilterFieldConfig,
  rolesListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/authorization/roles/hooks/use-filters";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  usePaginatedListQuery,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import type { RolesQueryPayload, RolesSortField } from "../schema/roles.schema";

// Mirrors DEFAULT_LIMIT in query.ts — kept here to avoid importing the server-side router
const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 100;

// Maps TanStack column IDs to server sort field names (and reverse)
const COLUMN_ID_TO_SORT_FIELD: Record<string, RolesSortField> = {
  role: "name",
  last_updated: "lastUpdated",
  assignedKeys: "assignedKeys",
  permissions: "assignedPermissions",
};
const SORT_FIELD_TO_COLUMN_ID: Record<RolesSortField, string> = {
  name: "role",
  lastUpdated: "last_updated",
  assignedKeys: "assignedKeys",
  assignedPermissions: "permissions",
};

type RolesFilterParams = Pick<
  RolesQueryPayload,
  "name" | "description" | "keyName" | "keyId" | "permissionSlug" | "permissionName"
>;

type RolesResponse = { roles: RoleBasic[]; total: number };

export function useRolesListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const result = usePaginatedListQuery<
    RolesResponse,
    RolesFilterValue,
    RolesSortField,
    RolesFilterParams
  >({
    pageSize,
    defaultPageSize: DEFAULT_PAGE_SIZE,
    maxPageSize: MAX_PAGE_SIZE,
    defaultSortField: "lastUpdated",
    columnIdToSortField: COLUMN_ID_TO_SORT_FIELD,
    sortFieldToColumnId: SORT_FIELD_TO_COLUMN_ID,
    useFilters,
    filterFieldNames: rolesListFilterFieldNames,
    filterFieldConfig: rolesFilterFieldConfig,
    useListQuery: (params) =>
      trpc.authorization.roles.query.useQuery(params, PAGINATED_LIST_QUERY_OPTIONS),
    usePrefetchNextPage: () => {
      const utils = trpc.useUtils();
      return (params) =>
        utils.authorization.roles.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS);
    },
  });

  return {
    roles: result.data?.roles ?? [],
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
