import {
  permissionsFilterFieldConfig,
  permissionsListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import type { PermissionsFilterValue } from "@/app/(app)/[workspaceSlug]/authorization/permissions/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/authorization/permissions/hooks/use-filters";
import { trpc } from "@/lib/trpc/client";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import type { PermissionsQueryPayload } from "../schema/permissions.schema";

const PREFETCH_PAGES_AHEAD = 2;

type PermissionsFilterParams = Pick<
  PermissionsQueryPayload,
  "name" | "description" | "slug" | "roleId" | "roleName"
>;

// Mirrors DEFAULT_LIMIT in query.ts
const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;

function buildQueryParams(filters: PermissionsFilterValue[]): PermissionsFilterParams {
  const params: PermissionsFilterParams = {
    name: [],
    description: [],
    slug: [],
    roleId: [],
    roleName: [],
  };

  for (const filter of filters) {
    if (!permissionsListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
      continue;
    }

    const fieldConfig = permissionsFilterFieldConfig[filter.field];
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

export function usePermissionsListPaginated(pageSize = DEFAULT_PAGE_SIZE) {
  const normalizedPageSize =
    Number.isFinite(pageSize) && pageSize > 0
      ? Math.min(Math.floor(pageSize), MAX_PAGE_SIZE)
      : DEFAULT_PAGE_SIZE;

  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  // Stable string key derived from filter content
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

  const baseParams = useMemo<PermissionsFilterParams>(() => buildQueryParams(filters), [filters]);

  const queryParams = useMemo(
    () => ({
      ...baseParams,
      page: normalizedPage,
      limit: normalizedPageSize,
    }),
    [baseParams, normalizedPage, normalizedPageSize],
  );

  const utils = trpc.useUtils();

  const { data, isLoading, isFetching } = trpc.authorization.permissions.query.useQuery(
    queryParams,
    {
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
      keepPreviousData: true,
    },
  );

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
      utils.authorization.permissions.query.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [normalizedPage, totalPages, queryParams, utils.authorization.permissions.query]);

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
    permissions: data?.permissions ?? [],
    isLoading,
    isInitialLoading,
    isPending: isFetching,
    isFetching,
    page: normalizedPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
  };
}
