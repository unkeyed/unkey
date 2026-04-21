import {
  type SortUrlValue,
  parseAsSortArray,
} from "@/components/logs/validation/utils/nuqs-parsers";
import type { SortingState } from "@tanstack/react-table";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";

const PREFETCH_PAGES_AHEAD = 2;

// Shared tRPC options — cached-forever paginated lists use the same defaults.
export const PAGINATED_LIST_QUERY_OPTIONS = {
  staleTime: Number.POSITIVE_INFINITY,
  refetchOnMount: false,
  refetchOnWindowFocus: false,
  keepPreviousData: true,
} as const;

export const PAGINATED_LIST_PREFETCH_OPTIONS = {
  staleTime: Number.POSITIVE_INFINITY,
} as const;

type FilterLike = {
  field: string;
  operator: string;
  value: unknown;
};

type FilterFieldConfig = {
  operators: readonly string[];
};

type PaginatedResponse = {
  total: number;
};

export type PageSortQueryParams<TSortField extends string> = {
  page: number;
  limit: number;
  sortBy: TSortField;
  sortOrder: "asc" | "desc";
};

type FilterParamsConstraint = Record<
  string,
  { operator: string; value: string }[] | null | undefined
>;

export type PaginatedListConfig<
  TResponse extends PaginatedResponse,
  TFilter extends FilterLike,
  TSortField extends string,
  TFilterParams extends FilterParamsConstraint,
> = {
  pageSize: number;
  defaultPageSize: number;
  maxPageSize: number;
  defaultSortField: TSortField;
  defaultSortDirection?: "asc" | "desc";
  columnIdToSortField: Record<string, TSortField>;
  sortFieldToColumnId: Record<TSortField, string>;
  useFilters: () => { filters: TFilter[] };
  filterFieldNames: readonly string[];
  filterFieldConfig: Record<string, FilterFieldConfig>;
  useListQuery: (params: TFilterParams & PageSortQueryParams<TSortField>) => {
    data: TResponse | undefined;
    isLoading: boolean;
    isFetching: boolean;
  };
  usePrefetchNextPage: () => (params: TFilterParams & PageSortQueryParams<TSortField>) => void;
};

// Shared backbone for server-paginated list views (roles, permissions, ...).
// Owns URL-synced `page` and `sort` state, translates filter hook output into
// tRPC query params, clamps the page to totals, and prefetches the next few
// pages so navigation feels instant. Callers supply the filter hook, the list
// query, and the prefetch helper so feature-specific types flow through.
export function usePaginatedListQuery<
  TResponse extends PaginatedResponse,
  TFilter extends FilterLike,
  TSortField extends string,
  TFilterParams extends FilterParamsConstraint,
>(config: PaginatedListConfig<TResponse, TFilter, TSortField, TFilterParams>) {
  const {
    pageSize,
    defaultPageSize,
    maxPageSize,
    defaultSortField,
    defaultSortDirection = "desc",
    columnIdToSortField,
    sortFieldToColumnId,
    useFilters,
    filterFieldNames,
    filterFieldConfig,
    useListQuery,
    usePrefetchNextPage,
  } = config;

  const defaultSortParams = useMemo<SortUrlValue<TSortField>[]>(
    () => [{ column: defaultSortField, direction: defaultSortDirection }],
    [defaultSortField, defaultSortDirection],
  );

  const normalizedPageSize =
    Number.isFinite(pageSize) && pageSize > 0
      ? Math.min(Math.floor(pageSize), maxPageSize)
      : defaultPageSize;

  const { filters } = useFilters();
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);
  const [sortParams, setSortParams] = useQueryState("sort", parseAsSortArray<TSortField>());

  // Ensure the default sort is always reflected in the URL.
  const effectiveSortParams = sortParams && sortParams.length > 0 ? sortParams : defaultSortParams;

  useEffect(() => {
    if (!sortParams || sortParams.length === 0) {
      setSortParams(defaultSortParams);
    }
  }, [sortParams, setSortParams, defaultSortParams]);

  // Keep only the first URL-derived sort entry whose column is an own key of
  // the caller's allowed set, falling back to defaults otherwise. The server
  // honors a single sortBy/sortOrder, so collapsing to one entry keeps the
  // table UI state and the tRPC query in sync. hasOwnProperty.call avoids
  // treating inherited Object.prototype methods (toString, hasOwnProperty…)
  // as valid columns when a crafted URL references them.
  const validSortParams = useMemo<SortUrlValue<TSortField>[]>(() => {
    const firstValid = effectiveSortParams.find((s) =>
      Object.prototype.hasOwnProperty.call(sortFieldToColumnId, s.column),
    );
    return firstValid ? [firstValid] : defaultSortParams;
  }, [effectiveSortParams, sortFieldToColumnId, defaultSortParams]);

  const sorting: SortingState = useMemo(() => {
    return validSortParams.map((s) => ({
      id: sortFieldToColumnId[s.column],
      desc: s.direction === "desc",
    }));
  }, [validSortParams, sortFieldToColumnId]);

  const onSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      const firstValid = next.find((s) =>
        Object.prototype.hasOwnProperty.call(columnIdToSortField, s.id),
      );
      const mapped: SortUrlValue<TSortField>[] = firstValid
        ? [
            {
              column: columnIdToSortField[firstValid.id],
              direction: firstValid.desc ? "desc" : "asc",
            },
          ]
        : defaultSortParams;
      setSortParams(mapped);
      setPage(1);
    },
    [sorting, setSortParams, setPage, columnIdToSortField, defaultSortParams],
  );

  // Stable string key from filter content — prevents spurious page resets when
  // the filter hook returns a new array reference for the same values.
  const filtersKey = useMemo(
    () => filters.map((f) => `${f.field}:${f.operator}:${String(f.value)}`).join("|"),
    [filters],
  );

  // Reset to page 1 only when filter content actually changes, not on mount.
  // The useEffect below syncs URL state for subsequent renders, but the render
  // that observes the filter change still sees the old normalizedPage — without
  // queryPage below, that render would fire one stale request for the previous
  // page against the new filters before setPage(1) commits. The null guard
  // keeps first-mount URL-persisted pages intact; we only override on a real
  // filter transition.
  const prevFiltersKeyRef = useRef<string | null>(null);
  const queryPage =
    prevFiltersKeyRef.current !== null && filtersKey !== prevFiltersKeyRef.current
      ? 1
      : normalizedPage;
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

  const filterParams = useMemo<TFilterParams>(() => {
    const params = Object.fromEntries(
      filterFieldNames.map((name) => [name, []]),
    ) as unknown as TFilterParams;
    for (const filter of filters) {
      if (!filterFieldNames.includes(filter.field)) {
        continue;
      }
      const bucket = params[filter.field];
      if (!bucket) {
        continue;
      }
      const fieldConfig = filterFieldConfig[filter.field];
      if (!fieldConfig || !fieldConfig.operators.includes(filter.operator)) {
        continue;
      }
      if (typeof filter.value === "string") {
        bucket.push({
          operator: filter.operator,
          value: filter.value,
        });
      }
    }
    return params;
  }, [filters, filterFieldNames, filterFieldConfig]);

  const queryParams = useMemo(
    () =>
      ({
        ...filterParams,
        page: queryPage,
        limit: normalizedPageSize,
        sortBy: validSortParams[0].column,
        sortOrder: validSortParams[0].direction,
      }) as TFilterParams & PageSortQueryParams<TSortField>,
    [filterParams, queryPage, normalizedPageSize, validSortParams],
  );

  const { data, isLoading, isFetching } = useListQuery(queryParams);
  const prefetchPage = usePrefetchNextPage();

  const isInitialLoading = isLoading && !data;
  const totalCount = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / normalizedPageSize));

  // Clamp page to valid range after data/totalPages updates.
  useEffect(() => {
    if (data == null) {
      return;
    }
    if (queryPage > totalPages) {
      setPage(totalPages);
    }
  }, [data, queryPage, totalPages, setPage]);

  // Prefetch the next few pages so navigation feels instant.
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = queryPage + i;
      if (nextPage > totalPages) {
        break;
      }
      prefetchPage({ ...queryParams, page: nextPage });
    }
  }, [queryPage, totalPages, prefetchPage, queryParams]);

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
    data,
    isInitialLoading,
    isFetching,
    page: queryPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
