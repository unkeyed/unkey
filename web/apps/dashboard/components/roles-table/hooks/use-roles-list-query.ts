import {
  type RolesFilterValue,
  rolesFilterFieldConfig,
  rolesListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/roles/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/authorization/roles/hooks/use-filters";
import type { RolesQueryPayload } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/table/query-logs.schema";
import type { RolesSortField } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/table/query-logs.schema";
import { parseAsSortArray } from "@/components/logs/validation/utils/nuqs-parsers";
import { trpc } from "@/lib/trpc/client";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";

const PREFETCH_PAGES_AHEAD = 2;

type RolesFilterParams = Pick<
  RolesQueryPayload,
  "name" | "description" | "keyName" | "keyId" | "permissionSlug" | "permissionName"
>;

// Mirrors DEFAULT_LIMIT in query.ts — kept here to avoid importing the server-side router
const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;

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

function buildQueryParams(filters: RolesFilterValue[]): RolesFilterParams {
  const params = Object.fromEntries(
    rolesListFilterFieldNames.map((field) => [field, []]),
  ) as RolesFilterParams;

  for (const filter of filters) {
    if (!rolesListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
      continue;
    }

    const fieldConfig = rolesFilterFieldConfig[filter.field];
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

export function useRolesListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const normalizedPageSize =
    Number.isFinite(pageSize) && pageSize > 0
      ? Math.min(Math.floor(pageSize), MAX_PAGE_SIZE)
      : DEFAULT_PAGE_SIZE;

  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);
  const [sortParams, setSortParams] = useQueryState("sort", parseAsSortArray<RolesSortField>());

  const sorting: SortingState = useMemo(() => {
    if (!sortParams || sortParams.length === 0) {
      return [{ id: "last_updated", desc: true }];
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
  // useQueryStates returns a new array reference for the same filter values.
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

  const baseParams = useMemo<RolesFilterParams>(() => buildQueryParams(filters), [filters]);

  const queryParams = useMemo(
    () => ({
      ...baseParams,
      page: normalizedPage,
      limit: normalizedPageSize,
      sortBy: sortParams?.[0]?.column ?? "lastUpdated",
      sortOrder: sortParams?.[0]?.direction ?? "desc",
    }),
    [baseParams, normalizedPage, normalizedPageSize, sortParams],
  );

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.authorization.roles.query.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    keepPreviousData: true,
  });

  const isInitialLoading = isLoading && !data;

  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / normalizedPageSize));

  // Clamp page to valid range after data/totalPages updates.
  useEffect(() => {
    if (normalizedPage > totalPages) {
      setPage(totalPages);
    }
  }, [normalizedPage, totalPages, setPage]);

  // Prefetch the next few pages so navigation feels instant.
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = normalizedPage + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.authorization.roles.query.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [normalizedPage, totalPages, queryParams, utils.authorization.roles.query]);

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
    roles: data?.roles ?? [],
    isLoading,
    isInitialLoading,
    isPending: isFetching,
    isFetching,
    page: normalizedPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
