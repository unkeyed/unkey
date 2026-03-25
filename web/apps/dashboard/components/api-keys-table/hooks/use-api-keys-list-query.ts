import {
  keysListFilterFieldConfig,
  keysListFilterFieldNames,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/hooks/use-filters";
import { trpc } from "@/lib/trpc/client";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import type { ApiKeysQueryPayload } from "../schema/api-keys.schema";

const DEFAULT_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 200;
const PREFETCH_PAGES_AHEAD = 2;

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

  // Reset to page 1 when filters change, but not on initial mount
  const filtersKey = useMemo(
    () => filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|"),
    [filters],
  );
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

  const queryParams = useMemo(() => {
    const params: ApiKeysQueryPayload = {
      limit: normalizedPageSize,
      page: normalizedPage,
      ...Object.fromEntries(keysListFilterFieldNames.map((field) => [field, []])),
      keyAuthId,
    };

    for (const filter of filters) {
      if (!keysListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        continue;
      }

      const fieldConfig = keysListFilterFieldConfig[filter.field];
      if (!fieldConfig.operators.includes(filter.operator)) {
        throw new Error("Invalid operator");
      }

      if (typeof filter.value === "string") {
        params[filter.field]?.push({
          operator: filter.operator,
          value: filter.value,
        });
      }
    }

    return params;
  }, [filters, keyAuthId, normalizedPage, normalizedPageSize]);

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

  // Clamp page to valid range after data loads
  useEffect(() => {
    if (normalizedPage > totalPages) {
      setPage(totalPages);
    }
  }, [normalizedPage, totalPages, setPage]);

  // Prefetch adjacent pages for instant navigation
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = normalizedPage + i;
      if (nextPage > totalPages) {
        break;
      }
      utils.api.keys.list.prefetch(
        { ...queryParams, page: nextPage },
        { staleTime: Number.POSITIVE_INFINITY },
      );
    }
  }, [normalizedPage, totalPages, queryParams, utils.api.keys.list]);

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
    keys: data?.keys ?? [],
    isLoading,
    isInitialLoading,
    isFetching,
    page: normalizedPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
  };
}
