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

// ---------------------------------------------------------------------------
// Primitives
//
// The two hooks below hold the machinery every server-paginated list view in
// the dashboard used to copy by hand: URL-synced `page` state, reset-to-page-1
// when the query inputs change, the deep-link clamp guard (ENG-2930), the
// adjacent-page prefetch, and the bounds-checked `onPageChange`. Feature hooks
// that share the shared sort/filter shape use `usePaginatedListQuery` below;
// hooks with a bespoke sort surface, time window, or realtime polling compose
// these two primitives directly and keep their feature-specific logic.
// ---------------------------------------------------------------------------

// Owns the URL `page` param and the reset-to-page-1 transition. `resetKey` is a
// stable string derived from every input the current OFFSET depends on (filter
// content, search, sort, query time…). When it changes, the page resets to 1.
//
// The returned `page` is already 1 on the render that first observes a
// resetKey change, before the effect below commits setPage(1). Without that,
// the same render would fire one stale request for the previous page against
// the new inputs. The null guard keeps a first-mount URL-persisted page intact;
// we only override on a real transition.
export function usePaginatedPage(resetKey: string) {
  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  const prevResetKeyRef = useRef<string | null>(null);
  const queryPage =
    prevResetKeyRef.current !== null && resetKey !== prevResetKeyRef.current ? 1 : normalizedPage;

  useEffect(() => {
    if (prevResetKeyRef.current === null) {
      prevResetKeyRef.current = resetKey;
      return;
    }
    if (resetKey !== prevResetKeyRef.current) {
      prevResetKeyRef.current = resetKey;
      setPage(1);
    }
  }, [resetKey, setPage]);

  return { page: queryPage, setPage };
}

type UsePaginatedNavigationParams<TData, TParams extends { page: number }> = {
  data: TData | undefined | null;
  page: number;
  totalPages: number;
  setPage: (page: number) => void;
  // The current query params. Prefetch requests reuse these with `page`
  // overridden. Pass the memoized object: the prefetch effect keys off its
  // identity, so it re-warms adjacent pages when the query shape (sort,
  // filters, time window) changes even while page and totalPages hold steady.
  queryParams: TParams;
  // Warm a page's query. Fresh identity each render is fine — a ref stabilizes
  // the effect, so callers do not need to memoize this.
  prefetch: (params: TParams) => void;
  prefetchPagesAhead?: number;
};

// Owns the clamp guard, the adjacent-page prefetch, and `onPageChange`. Kept
// separate from `usePaginatedPage` so a caller can compute `data`/`totalPages`
// from its own query (and run any other hooks it needs) in between.
export function usePaginatedNavigation<TData, TParams extends { page: number }>({
  data,
  page,
  totalPages,
  setPage,
  queryParams,
  prefetch,
  prefetchPagesAhead = PREFETCH_PAGES_AHEAD,
}: UsePaginatedNavigationParams<TData, TParams>) {
  // Clamp page to valid range after data loads. The data guard keeps a
  // deep-linked page (e.g. ?page=3) from snapping to 1 on first render, when
  // totalCount is still 0 and totalPages collapses to 1 (ENG-2930).
  //
  // The overview hooks additionally gated this on `!isFetching` (#6560): they
  // fed the pre-reset page in here, so the clamp could pair a stale totalPages
  // with a page belonging to the previous result set. usePaginatedPage's
  // synchronous reset closes that window, so no isFetching gate is needed.
  useEffect(() => {
    if (data == null) {
      return;
    }
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [data, page, totalPages, setPage]);

  // Prefetch the next few pages so navigation feels instant. A ref keeps a
  // fresh caller arrow each render from re-firing the effect; the effect re-runs
  // on page/totalPages changes and whenever queryParams identity changes.
  const prefetchRef = useRef(prefetch);
  prefetchRef.current = prefetch;
  useEffect(() => {
    for (let i = 1; i <= prefetchPagesAhead; i++) {
      const nextPage = page + i;
      if (nextPage > totalPages) {
        break;
      }
      prefetchRef.current({ ...queryParams, page: nextPage });
    }
  }, [page, totalPages, prefetchPagesAhead, queryParams]);

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );

  return { onPageChange };
}

// Derive totalPages from a total count and page size, never below 1.
export function computeTotalPages(totalCount: number, pageSize: number) {
  return Math.max(1, Math.ceil(totalCount / pageSize));
}

// Clamp a caller-supplied page size into [1, maxPageSize], falling back to the
// default for non-finite or non-positive input.
export function normalizePageSize(pageSize: number, defaultPageSize: number, maxPageSize: number) {
  return Number.isFinite(pageSize) && pageSize > 0
    ? Math.min(Math.floor(pageSize), maxPageSize)
    : defaultPageSize;
}

// Stable identity string for a set of filters. Feed it (or an extension of it,
// e.g. with a time window appended) as the page reset key so page state resets
// only when filter content actually changes — not when the filter hook returns
// a new array reference for the same values.
//
// Encoded as JSON tuples rather than delimiter-joined text so the result is
// unambiguous: a filter value containing a separator character cannot make two
// distinct states collapse to the same key (which would suppress a real page
// reset). The string is only ever compared for equality, never parsed.
export function paginationFilterKey(
  filters: ReadonlyArray<{ field: string; operator: string; value: unknown }>,
) {
  return JSON.stringify(filters.map((f) => [f.field, f.operator, f.value]));
}

// ---------------------------------------------------------------------------
// usePaginatedListQuery — the full shared hook for the common shape:
// URL `page` + URL `sort`, string-bucket filters via a filter hook, a single
// server sortBy/sortOrder, computed totalPages.
// ---------------------------------------------------------------------------

type FilterLike = {
  field: string;
  operator: string;
  value: unknown;
};

type FilterFieldConfig = {
  operators: readonly string[];
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
  TResponse,
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
  // Fresh identity each render is fine — the hook stabilizes via a ref so the
  // prefetch effect does not re-fire on every caller re-render. Callers do not
  // need to wrap this in useCallback.
  prefetch: (params: TFilterParams & PageSortQueryParams<TSortField>) => void;
  // Read the total row count off the response. Responses spell this differently
  // (`total`, `totalCount`, …), so callers map it explicitly.
  getTotalCount: (data: TResponse) => number;
  // Optional: when the URL has no `sort`, write the default into it on mount.
  // Defaults to true. Set false to keep the URL clean until the user sorts
  // (the table still renders and queries the default sort locally).
  syncDefaultSortToUrl?: boolean;
};

// Shared backbone for server-paginated list views (roles, permissions, ...).
// Owns URL-synced `page` and `sort` state, translates filter hook output into
// tRPC query params, clamps the page to totals, and prefetches the next few
// pages so navigation feels instant. Callers supply the filter hook, the list
// query, and the prefetch helper so feature-specific types flow through.
export function usePaginatedListQuery<
  TResponse,
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
    prefetch,
    getTotalCount,
    syncDefaultSortToUrl = true,
  } = config;

  const defaultSortParams = useMemo<SortUrlValue<TSortField>[]>(
    () => [{ column: defaultSortField, direction: defaultSortDirection }],
    [defaultSortField, defaultSortDirection],
  );

  const normalizedPageSize = normalizePageSize(pageSize, defaultPageSize, maxPageSize);

  const { filters } = useFilters();
  const [sortParams, setSortParams] = useQueryState("sort", parseAsSortArray<TSortField>());

  // Fall back to the default sort when the URL has none. `effectiveSortParams`
  // drives the local table/query regardless; the effect only writes it back to
  // the URL when syncDefaultSortToUrl is set.
  const effectiveSortParams = sortParams && sortParams.length > 0 ? sortParams : defaultSortParams;

  useEffect(() => {
    if (!syncDefaultSortToUrl) {
      return;
    }
    if (!sortParams || sortParams.length === 0) {
      setSortParams(defaultSortParams);
    }
  }, [sortParams, setSortParams, defaultSortParams, syncDefaultSortToUrl]);

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

  const filtersKey = useMemo(() => paginationFilterKey(filters), [filters]);

  const { page: queryPage, setPage } = usePaginatedPage(filtersKey);

  const onSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      const firstValid = next.find((s) =>
        Object.prototype.hasOwnProperty.call(columnIdToSortField, s.id),
      );
      if (firstValid) {
        setSortParams([
          {
            column: columnIdToSortField[firstValid.id],
            direction: firstValid.desc ? "desc" : "asc",
          },
        ]);
      } else {
        // Sorting was cleared (or an unknown column). Mirror syncDefaultSortToUrl:
        // hooks that keep the URL clean clear the param, others pin the default so
        // the mount effect does not immediately rewrite it.
        setSortParams(syncDefaultSortToUrl ? defaultSortParams : null);
      }
      setPage(1);
    },
    [sorting, setSortParams, setPage, columnIdToSortField, defaultSortParams, syncDefaultSortToUrl],
  );

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

  const isInitialLoading = isLoading && !data;
  const totalCount = data ? getTotalCount(data) : 0;
  const totalPages = computeTotalPages(totalCount, normalizedPageSize);

  const { onPageChange } = usePaginatedNavigation({
    data,
    page: queryPage,
    totalPages,
    setPage,
    queryParams,
    prefetch,
  });

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
