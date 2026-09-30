import {
  rootKeysFilterFieldConfig,
  rootKeysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import type { RootKeysFilterValue } from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/settings/root-keys/hooks/use-filters";
import {
  PAGINATED_LIST_QUERY_OPTIONS,
  usePaginatedListQuery,
} from "@/hooks/use-paginated-list-query";
import { ROOT_KEYS_V2_QUERY_KEY, type V2RootKey, listRootKeys } from "@/lib/root-keys-api";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { useWorkspace } from "@/providers/workspace-provider";
import { useQuery } from "@tanstack/react-query";
import type { RootKeysQueryPayload, RootKeysSortField } from "../schema/query-logs.schema";

const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;

const COLUMN_ID_TO_SORT_FIELD: Record<string, RootKeysSortField> = {
  root_key: "name",
  created_at: "createdAt",
  last_used: "lastUsedAt",
  last_updated: "lastUpdatedAt",
};

const SORT_FIELD_TO_COLUMN_ID: Record<RootKeysSortField, string> = {
  name: "root_key",
  createdAt: "created_at",
  lastUsedAt: "last_used",
  lastUpdatedAt: "last_updated",
};

type RootKeysFilterParams = Pick<RootKeysQueryPayload, "name" | "start" | "permission">;
type RootKeysResponse = { keys: RootKey[]; total: number };
type SelectParams = RootKeysFilterParams & {
  page: number;
  limit: number;
  sortBy: RootKeysSortField;
  sortOrder: "asc" | "desc";
};

function matches(
  value: string | null,
  filters: { operator: string; value: string }[] | null | undefined,
) {
  if (!filters?.length) {
    return true;
  }
  const normalized = (value ?? "").toLowerCase();
  return filters.some((filter) => {
    const expected = filter.value.toLowerCase();
    return filter.operator === "is" ? normalized === expected : normalized.includes(expected);
  });
}

function toRootKey(rootKey: V2RootKey): RootKey {
  const prefixEnd = rootKey.start.indexOf("_");
  const permissions = rootKey.permissions.map((permission) => ({
    id: permission,
    name: permission,
  }));

  return {
    id: rootKey.keyId,
    prefix: prefixEnd === -1 ? "" : rootKey.start.slice(0, prefixEnd),
    keyAuthId: rootKey.keyId,
    start: prefixEnd === -1 ? rootKey.start : rootKey.start.slice(prefixEnd + 1),
    end: rootKey.end,
    createdAt: rootKey.createdAt,
    lastUsedAt: rootKey.lastUsedAt,
    lastUpdatedAt: null,
    expires: rootKey.expires,
    name: rootKey.name,
    permissionSummary: {
      total: permissions.length,
      categories: {},
      hasCriticalPerm: permissions.some(({ name }) =>
        ["delete", "decrypt", "remove"].some((action) => name.toLowerCase().includes(action)),
      ),
    },
    permissions,
  };
}

export function selectRootKeysPage(rootKeys: V2RootKey[], params: SelectParams): RootKeysResponse {
  const filtered = rootKeys
    .filter(
      (rootKey) =>
        matches(rootKey.name, params.name) &&
        matches(rootKey.start, params.start) &&
        matches(rootKey.permissions.join("\n"), params.permission),
    )
    .map(toRootKey);

  filtered.sort((left, right) => {
    const leftValue = left[params.sortBy] ?? "";
    const rightValue = right[params.sortBy] ?? "";
    const comparison =
      typeof leftValue === "number" && typeof rightValue === "number"
        ? leftValue - rightValue
        : String(leftValue).localeCompare(String(rightValue));
    return (comparison || left.id.localeCompare(right.id)) * (params.sortOrder === "asc" ? 1 : -1);
  });

  const offset = (params.page - 1) * params.limit;
  return {
    keys: filtered.slice(offset, offset + params.limit),
    total: filtered.length,
  };
}

export function useRootKeysV2ListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const { workspace } = useWorkspace();
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
    useListQuery: (params) => {
      // biome-ignore lint/correctness/useHookAtTopLevel: hook factory invoked unconditionally inside the paginated-list hook
      const query = useQuery({
        queryKey: [...ROOT_KEYS_V2_QUERY_KEY, workspace?.id],
        queryFn: listRootKeys,
        ...PAGINATED_LIST_QUERY_OPTIONS,
        enabled: workspace !== null,
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
