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
// Shared machinery for every server-paginated list view. Two stateful hooks:
// `usePaginatedPage` (URL-synced `page` plus reset-to-page-1 when the query
// inputs change) and `usePaginatedNavigation` (deep-link clamp guard,
// adjacent-page prefetch, bounds-checked `onPageChange`). Plus the count
// helpers (`computeTotalPages`, `computeFallbackTotalPages`,
// `useFallbackTotalPages`, `normalizePageSize`) and reset-key builders
// (`paginationFilterKey`, `paginationSortKey`). Hooks with the common
// sort/filter shape use `usePaginatedListQuery` below; hooks with a bespoke
// sort surface, time window, or polling compose these primitives directly.
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

type UsePaginatedNavigationParams<TParams extends { page: number }> = {
  data: unknown;
  page: number;
  totalPages: number;
  setPage: (page: number) => void;
  // Raw query flags, used only to derive the two loading states every list view
  // needs. Required rather than defaulted: a missing flag would silently pin
  // `isInitialLoading`/`isNavigating` to false and the caller would render a
  // list that never reports loading.
  isLoading: boolean;
  isFetching: boolean;
  // The current query params. Prefetch requests reuse these with `page`
  // overridden. Pass the memoized object: the prefetch effect keys off its
  // identity, so it re-warms adjacent pages when the query shape (sort,
  // filters, time window) changes even while page and totalPages hold steady.
  queryParams: TParams;
  // Warm a page's query. Fresh identity each render is fine — a ref stabilizes
  // the effect, so callers do not need to memoize this.
  prefetch: (params: TParams) => void;
  // Extra identity for values the `prefetch` closure captures but `queryParams`
  // does not carry (e.g. a keyspace id). Without it the prefetch effect cannot
  // see those values change and never re-warms the adjacent pages.
  prefetchKey?: string;
  // Set false while the view pins itself to page 1 (live-tail modes), where
  // clamping the pinned page against a live total would fight the pin. Suspends
  // the clamp ONLY: the prefetch keeps running so pages are warm when the user
  // leaves live tail, and `onPageChange` still validates against totalPages.
  // Defaults to true.
  clampEnabled?: boolean;
};

// Owns the clamp guard, the adjacent-page prefetch, and `onPageChange`. Kept
// separate from `usePaginatedPage` so a caller can compute `data`/`totalPages`
// from its own query (and run any other hooks it needs) in between.
export function usePaginatedNavigation<TParams extends { page: number }>({
  data,
  page,
  totalPages,
  setPage,
  isLoading,
  isFetching,
  queryParams,
  prefetch,
  prefetchKey,
  clampEnabled = true,
}: UsePaginatedNavigationParams<TParams>) {
  // Clamp page to valid range after data loads. The data guard keeps a
  // deep-linked page (e.g. ?page=3) from snapping to 1 on first render, when
  // totalCount is still 0 and totalPages collapses to 1. No isFetching gate:
  // usePaginatedPage's synchronous reset already surfaces page 1 before a stale
  // totalPages from the previous result set can pair with the pre-reset page.
  useEffect(() => {
    if (!clampEnabled || data == null) {
      return;
    }
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [clampEnabled, data, page, totalPages, setPage]);

  // Prefetch the next few pages so navigation feels instant. A ref keeps a
  // fresh caller arrow each render from re-firing the effect; the effect re-runs
  // on page/totalPages changes, whenever queryParams identity changes, and
  // whenever prefetchKey covers a closure value queryParams does not carry.
  const prefetchRef = useRef(prefetch);
  prefetchRef.current = prefetch;
  // biome-ignore lint/correctness/useExhaustiveDependencies: prefetchKey is read through the caller's prefetch closure, not this body, so it has to be listed to re-warm on change
  useEffect(() => {
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = page + i;
      if (nextPage > totalPages) {
        break;
      }
      prefetchRef.current({ ...queryParams, page: nextPage });
    }
  }, [page, totalPages, queryParams, prefetchKey]);

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );

  // Derived here so every list view agrees on what "still loading" versus
  // "navigating between pages" means. Under keepPreviousData a page change
  // keeps the previous rows on screen, so `isFetching` alone cannot tell the
  // two apart — only the first load has no data to show.
  const isInitialLoading = isLoading && !data;
  const isNavigating = isFetching && !isInitialLoading;

  return { onPageChange, isInitialLoading, isNavigating };
}

// Derive totalPages from a total count and page size, never below 1.
export function computeTotalPages(totalCount: number, pageSize: number) {
  return Math.max(1, Math.ceil(totalCount / pageSize));
}

// Derive totalPages when the server could not return a total count (e.g. a
// failed ClickHouse count query). While the current page has rows, advertise
// one page ahead only if it is full, so Next stays available exactly as far as
// data has been proven to exist. An out-of-range page is only detectable once
// it comes back empty; reporting the last page seen with rows as the total
// then lets the caller's clamp snap back to the last real page, or to page 1
// for a stale deep link that never saw data (lastNonEmptyPage stays 1 until a
// settled response has rows). The isFetching gate matters under
// keepPreviousData: without it, fetching the clamp target while the previous
// empty result is still displayed would re-trigger the collapse and cascade
// the clamp toward page 1.
export function computeFallbackTotalPages(args: {
  isFetching: boolean;
  hasData: boolean;
  pageRowCount: number;
  queryPage: number;
  limit: number;
  lastNonEmptyPage: number;
}) {
  const { isFetching, hasData, pageRowCount, queryPage, limit, lastNonEmptyPage } = args;
  const isEmptyPageBeyondFirst = !isFetching && hasData && pageRowCount === 0 && queryPage > 1;
  if (isEmptyPageBeyondFirst) {
    return Math.min(lastNonEmptyPage, queryPage - 1);
  }
  // Mid-fetch, keepPreviousData still reports the PREVIOUS page's rows, so the
  // `pageRowCount >= limit` test below would credit them to the page in flight
  // and advertise a page-ahead nothing has been observed at. A user clicking
  // Next twice in a row could then jump straight past the in-flight page — and
  // since that page is never observed it never reaches lastNonEmptyPage, so the
  // empty-page branch above later clamps back behind it, skipping a page that
  // did have rows. Hold the total at what is already proven until the response
  // settles; Next re-enables one commit later.
  if (isFetching) {
    return Math.max(queryPage, lastNonEmptyPage);
  }
  return queryPage + (pageRowCount >= limit ? 1 : 0);
}

// Stateful companion to computeFallbackTotalPages: owns the lastNonEmptyPage
// tracking so the count-outage rule lives in one place instead of being split
// between this module and the caller. `resetKey` is the caller's page reset key
// (filter content, time window…); when it changes the observed-pages memory is
// dropped, since rows seen under the old inputs say nothing about the new ones.
export function useFallbackTotalPages(args: {
  isFetching: boolean;
  hasData: boolean;
  pageRowCount: number;
  queryPage: number;
  limit: number;
  resetKey: string;
}) {
  const { isFetching, hasData, pageRowCount, queryPage, limit, resetKey } = args;

  const lastNonEmptyPageRef = useRef(1);

  // Reset during render, not in an effect: totalPages is derived below on this
  // same render, and usePaginatedPage has already snapped queryPage to 1 here.
  // Deferring to an effect would pair the new page with the old page memory. A
  // ref is safe despite React possibly discarding this render, because the
  // reset is idempotent: a re-render at the same resetKey re-derives the same
  // total.
  const prevResetKeyRef = useRef(resetKey);
  if (prevResetKeyRef.current !== resetKey) {
    prevResetKeyRef.current = resetKey;
    lastNonEmptyPageRef.current = 1;
  }

  // Same isFetching reasoning as above: rows on screen during a fetch belong to
  // the previous page and must not mark the in-flight page as non-empty.
  useEffect(() => {
    if (!isFetching && pageRowCount > 0) {
      lastNonEmptyPageRef.current = queryPage;
    }
  }, [isFetching, pageRowCount, queryPage]);

  return computeFallbackTotalPages({
    isFetching,
    hasData,
    pageRowCount,
    queryPage,
    limit,
    lastNonEmptyPage: lastNonEmptyPageRef.current,
  });
}

// Clamp a caller-supplied page size into [1, maxPageSize], falling back to the
// default for non-finite or non-positive input.
export function normalizePageSize(pageSize: number, defaultPageSize: number, maxPageSize: number) {
  // Math.max(1, …) after the floor: a fractional size like 0.5 clears the
  // `> 0` guard but floors to 0, which would make computeTotalPages divide by
  // zero and send `limit: 0` to the server.
  return Number.isFinite(pageSize) && pageSize > 0
    ? Math.min(Math.max(1, Math.floor(pageSize)), maxPageSize)
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

// Companion to paginationFilterKey for sort arrays, with the same
// unambiguous-encoding guarantee.
export function paginationSortKey(sorts: ReadonlyArray<{ column: string; direction: string }>) {
  return JSON.stringify(sorts.map((s) => [s.column, s.direction]));
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
  filterFieldNames: readonly (keyof TFilterParams & string)[];
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
  // Extra identity for values `useListQuery`/`prefetch` capture from their
  // closure rather than receiving through params (e.g. a keyspace id). Callers
  // that close over such a value must pass it here so prefetch re-warms the
  // adjacent pages when it changes.
  prefetchKey?: string;
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
    prefetchKey,
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

  // The server honors a single sortBy/sortOrder, so collapse a multi-sort URL
  // to its first valid entry. Keeping the rest would let `sorting` paint header
  // indicators for an ordering the server never applied — and callers that
  // re-sort rows client-side off `sorting` would disagree with the server
  // outright. hasOwnProperty.call avoids treating inherited Object.prototype
  // methods as valid columns when a crafted URL references them.
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
    // TS mandates the `unknown` step when casting to a generic type parameter;
    // the keyof constraint on filterFieldNames keeps the key set honest.
    const params = Object.fromEntries(
      filterFieldNames.map((name) => [name, []]),
    ) as unknown as TFilterParams;
    for (const filter of filters) {
      if (!filterFieldNames.some((name) => name === filter.field)) {
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

  const totalCount = data ? getTotalCount(data) : 0;
  const totalPages = computeTotalPages(totalCount, normalizedPageSize);

  const { onPageChange, isInitialLoading, isNavigating } = usePaginatedNavigation({
    data,
    page: queryPage,
    totalPages,
    setPage,
    isLoading,
    isFetching,
    queryParams,
    prefetch,
    prefetchKey,
  });

  return {
    data,
    // The two states list views actually render against. The raw `isFetching`
    // is deliberately not surfaced: every caller that had it re-derived
    // `isNavigating` from it by hand, which is the duplication this hook exists
    // to remove.
    isInitialLoading,
    isNavigating,
    page: queryPage,
    pageSize: normalizedPageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  };
}
