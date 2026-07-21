import { act, renderHook } from "@testing-library/react";
import { useCallback, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Controllable, in-memory stand-in for the URL query string. nuqs is otherwise
// kept real so the shared hook's `parseAsSortArray` parser still works; only
// `useQueryState` is swapped for a store-backed fake that updates synchronously
// (no nuqs throttle timers to flush in tests).
const urlStore: Record<string, unknown> = {};

vi.mock("nuqs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("nuqs")>();
  const useQueryState = (key: string, parser?: { defaultValue?: unknown }) => {
    const [, force] = useState(0);
    const read = () =>
      Object.prototype.hasOwnProperty.call(urlStore, key)
        ? urlStore[key]
        : (parser?.defaultValue ?? null);
    // biome-ignore lint/correctness/useExhaustiveDependencies: test-only fake; `read` closes over the stable `key`/`parser`.
    const setValue = useCallback(
      (next: unknown) => {
        urlStore[key] = typeof next === "function" ? next(read()) : next;
        force((n) => n + 1);
        return Promise.resolve(true);
      },
      [key],
    );
    return [read(), setValue];
  };
  return { ...actual, useQueryState };
});

import {
  computeFallbackTotalPages,
  normalizePageSize,
  paginationFilterKey,
  paginationSortKey,
  usePaginatedListQuery,
  usePaginatedNavigation,
  usePaginatedPage,
} from "./use-paginated-list-query";

// Mutable test doubles read through stable hook/function identities so the
// shared hook sees fresh values on each render after we mutate + rerender.
type TestResponse = { total: number };
let queryResult: {
  data: TestResponse | undefined;
  isLoading: boolean;
  isFetching: boolean;
};
let currentFilters: { field: string; operator: string; value: string }[];
let prefetchSpy: ReturnType<typeof vi.fn>;

function makeConfig(overrides?: { syncDefaultSortToUrl?: boolean }) {
  return {
    pageSize: 10,
    defaultPageSize: 10,
    maxPageSize: 100,
    defaultSortField: "createdAt" as const,
    columnIdToSortField: { created: "createdAt" } as Record<string, "createdAt">,
    sortFieldToColumnId: { createdAt: "created" } as Record<"createdAt", string>,
    useFilters: () => ({ filters: currentFilters }),
    filterFieldNames: [] as const,
    filterFieldConfig: {},
    useListQuery: () => queryResult,
    prefetch: prefetchSpy,
    getTotalCount: (data: TestResponse) => data.total,
    ...overrides,
  };
}

function render(overrides?: { syncDefaultSortToUrl?: boolean }) {
  return renderHook(() => usePaginatedListQuery(makeConfig(overrides)));
}

beforeEach(() => {
  for (const key of Object.keys(urlStore)) {
    delete urlStore[key];
  }
  queryResult = { data: undefined, isLoading: true, isFetching: true };
  currentFilters = [];
  prefetchSpy = vi.fn();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("usePaginatedListQuery", () => {
  describe("deep-link clamp guarantee (ENG-2930)", () => {
    it("does not clamp a deep-linked page to 1 before data loads", () => {
      // ?page=5 with no data yet: totalCount is 0 and totalPages collapses to 1,
      // but the data guard must keep the page intact until the first response.
      urlStore.page = 5;
      const { result } = render();

      expect(result.current.page).toBe(5);
    });

    it("keeps the deep-linked page once data confirms it is in range", () => {
      urlStore.page = 5;
      const { result, rerender } = render();
      expect(result.current.page).toBe(5);

      // 100 items / 10 per page = 10 pages, so page 5 is valid.
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      act(() => {
        rerender();
      });

      expect(result.current.page).toBe(5);
    });

    it("clamps to the last page once data shows the page is out of range", () => {
      urlStore.page = 5;
      const { result, rerender } = render();
      expect(result.current.page).toBe(5);

      // 20 items / 10 per page = 2 pages, so page 5 clamps down to 2.
      queryResult = { data: { total: 20 }, isLoading: false, isFetching: false };
      act(() => {
        rerender();
      });

      expect(result.current.page).toBe(2);
      expect(result.current.totalPages).toBe(2);
    });
  });

  describe("onPageChange", () => {
    beforeEach(() => {
      queryResult = { data: { total: 20 }, isLoading: false, isFetching: false };
    });

    it("ignores out-of-range targets", () => {
      const { result } = render();

      act(() => {
        result.current.onPageChange(5);
      });
      expect(result.current.page).toBe(1);

      act(() => {
        result.current.onPageChange(0);
      });
      expect(result.current.page).toBe(1);
    });

    it("navigates to an in-range page", () => {
      const { result } = render();

      act(() => {
        result.current.onPageChange(2);
      });
      expect(result.current.page).toBe(2);
    });
  });

  describe("prefetch", () => {
    it("prefetches the pages ahead and stops at the last page", () => {
      // 50 items / 10 per page = 5 pages; from page 1 it prefetches 2 and 3.
      queryResult = { data: { total: 50 }, isLoading: false, isFetching: false };
      urlStore.page = 1;
      render();

      // A page may be prefetched more than once across renders (idempotent;
      // react-query dedupes) — assert on the set of pages, not call order.
      const prefetchedPages = [...new Set(prefetchSpy.mock.calls.map((call) => call[0].page))].sort(
        (a, b) => a - b,
      );
      expect(prefetchedPages).toEqual([2, 3]);
    });

    it("does not prefetch past the last page", () => {
      // On the final page there is nothing ahead to prefetch.
      queryResult = { data: { total: 50 }, isLoading: false, isFetching: false };
      urlStore.page = 5;
      render();

      expect(prefetchSpy).not.toHaveBeenCalled();
    });
  });

  describe("filter changes", () => {
    it("preserves the URL page on first mount", () => {
      urlStore.page = 3;
      currentFilters = [{ field: "name", operator: "is", value: "a" }];
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      const { result } = render();

      expect(result.current.page).toBe(3);
    });

    it("resets to page 1 when filter content changes", () => {
      urlStore.page = 3;
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      const { result, rerender } = render();
      expect(result.current.page).toBe(3);

      currentFilters = [{ field: "name", operator: "is", value: "a" }];
      act(() => {
        rerender();
      });

      expect(result.current.page).toBe(1);
    });
  });

  describe("sorting", () => {
    it("falls back to the default sort when the URL has none", () => {
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      const { result } = render();

      expect(result.current.sorting).toEqual([{ id: "created", desc: true }]);
    });

    it("keeps every valid URL sort entry in sorting state, dropping unknown columns", () => {
      // A multi-sort deep link (produced by the pre-consolidation bespoke
      // hooks) must keep all its header indicators; only the first entry
      // drives the server query.
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      urlStore.sort = [
        { column: "createdAt", direction: "asc" },
        { column: "updatedAt", direction: "desc" },
        { column: "createdAt", direction: "desc" },
      ];
      const { result } = renderHook(() =>
        usePaginatedListQuery({
          ...makeConfig(),
          columnIdToSortField: { created: "createdAt", updated: "updatedAt" } as Record<
            string,
            "createdAt" | "updatedAt"
          >,
          sortFieldToColumnId: { createdAt: "created", updatedAt: "updated" } as Record<
            "createdAt" | "updatedAt",
            string
          >,
          defaultSortField: "createdAt" as const,
        }),
      );

      expect(result.current.sorting).toEqual([
        { id: "created", desc: false },
        { id: "updated", desc: true },
        { id: "created", desc: true },
      ]);
    });

    it("resets to page 1 when the sort changes", () => {
      urlStore.page = 3;
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      const { result } = render();
      expect(result.current.page).toBe(3);

      act(() => {
        result.current.onSortingChange([{ id: "created", desc: false }]);
      });

      expect(result.current.page).toBe(1);
      expect(result.current.sorting).toEqual([{ id: "created", desc: false }]);
    });

    it("clears the URL sort param when sorting is removed and syncDefaultSortToUrl is false", () => {
      // Hooks that opt into a clean URL (api-keys, root-keys) must not have the
      // default sort written back when the user removes sorting entirely.
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      urlStore.sort = [{ column: "createdAt", direction: "asc" }];
      const { result } = render({ syncDefaultSortToUrl: false });

      act(() => {
        result.current.onSortingChange([]);
      });

      expect(urlStore.sort).toBeNull();
    });

    it("pins the default sort in the URL when sorting is removed and syncDefaultSortToUrl is true", () => {
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      urlStore.sort = [{ column: "createdAt", direction: "asc" }];
      const { result } = render();

      act(() => {
        result.current.onSortingChange([]);
      });

      expect(urlStore.sort).toEqual([{ column: "createdAt", direction: "desc" }]);
    });
  });
});

describe("usePaginatedPage", () => {
  it("keeps the URL page on first mount (no reset)", () => {
    urlStore.page = 4;
    const { result } = renderHook(({ resetKey }) => usePaginatedPage(resetKey), {
      initialProps: { resetKey: "filters:a" },
    });

    expect(result.current.page).toBe(4);
  });

  it("clamps a non-positive URL page up to 1", () => {
    urlStore.page = -3;
    const { result } = renderHook(({ resetKey }) => usePaginatedPage(resetKey), {
      initialProps: { resetKey: "filters:a" },
    });

    expect(result.current.page).toBe(1);
  });

  it("resets to page 1 immediately on the render that observes a resetKey change", () => {
    urlStore.page = 4;
    const { result, rerender } = renderHook(({ resetKey }) => usePaginatedPage(resetKey), {
      initialProps: { resetKey: "filters:a" },
    });
    expect(result.current.page).toBe(4);

    // Changing the reset key (filters/sort/time changed) must surface page 1 in
    // the same render, before the effect commits it to the URL — otherwise a
    // stale request for page 4 fires against the new inputs.
    act(() => {
      rerender({ resetKey: "filters:b" });
    });

    expect(result.current.page).toBe(1);
    expect(urlStore.page).toBe(1);
  });
});

describe("usePaginatedPage + usePaginatedNavigation", () => {
  // Upstream (#6560) additionally gated the overview clamps on `!isFetching`,
  // because those hooks fed the *stale* page into the clamp on the render that
  // observed a filter change: with keepPreviousData, `totalPages` still described
  // the previous result set, so the clamp could snap the page to the old last
  // page before the reset committed. usePaginatedPage closes that window by
  // surfacing page 1 on that same render, so the clamp never sees a page drawn
  // from one result set alongside totalPages drawn from another. This test pins
  // that guarantee; if usePaginatedPage ever loses the synchronous reset, the
  // clamp regresses and this fails.
  it("does not clamp against stale totalPages when the reset key changes", () => {
    urlStore.page = 3;
    const { result, rerender } = renderHook(
      ({ resetKey, totalPages }) => {
        const { page, setPage } = usePaginatedPage(resetKey);
        usePaginatedNavigation({
          data: { total: 50 },
          page,
          totalPages,
          setPage,
          queryParams: { page },
          prefetch: vi.fn(),
        });
        return page;
      },
      { initialProps: { resetKey: "filters:a", totalPages: 5 } },
    );
    expect(result.current).toBe(3);

    // Filters changed; the response is in flight, so `data` and `totalPages`
    // still describe the previous result set (which had only 2 pages).
    act(() => {
      rerender({ resetKey: "filters:b", totalPages: 2 });
    });

    // Page 1, not the stale last page (2) that an ungated clamp would pick.
    expect(result.current).toBe(1);
    expect(urlStore.page).toBe(1);
  });
});

describe("usePaginatedNavigation", () => {
  it("does not clamp before data loads even if page is out of range", () => {
    const setPage = vi.fn();
    renderHook(() =>
      usePaginatedNavigation({
        data: undefined,
        page: 5,
        totalPages: 1,
        setPage,
        queryParams: { page: 5 },
        prefetch: vi.fn(),
      }),
    );

    expect(setPage).not.toHaveBeenCalled();
  });

  it("clamps to the last page once data shows the page is out of range", () => {
    const setPage = vi.fn();
    renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 5,
        totalPages: 2,
        setPage,
        queryParams: { page: 5 },
        prefetch: vi.fn(),
      }),
    );

    expect(setPage).toHaveBeenCalledWith(2);
  });

  it("prefetches the pages ahead (with query params) and stops at the last page", () => {
    const prefetch = vi.fn();
    renderHook(() =>
      usePaginatedNavigation({
        data: { total: 50 },
        page: 1,
        totalPages: 5,
        setPage: vi.fn(),
        // A representative query shape — prefetch must carry it, overriding page.
        queryParams: { page: 1, sortBy: "createdAt" },
        prefetch,
      }),
    );

    // Prefetch receives the full params with page overridden, not a bare page.
    expect(prefetch.mock.calls.every((call) => call[0].sortBy === "createdAt")).toBe(true);
    const prefetched = [...new Set(prefetch.mock.calls.map((call) => call[0].page))].sort(
      (a, b) => a - b,
    );
    expect(prefetched).toEqual([2, 3]);
  });

  it("does not prefetch past the last page", () => {
    const prefetch = vi.fn();
    renderHook(() =>
      usePaginatedNavigation({
        data: { total: 50 },
        page: 5,
        totalPages: 5,
        setPage: vi.fn(),
        queryParams: { page: 5 },
        prefetch,
      }),
    );

    expect(prefetch).not.toHaveBeenCalled();
  });

  it("re-prefetches when queryParams identity changes while page holds steady", () => {
    // Sorting on page 1 changes queryParams but not page/totalPages; the effect
    // must still re-warm the adjacent pages for the new query shape.
    const prefetch = vi.fn();
    const { rerender } = renderHook(
      ({ queryParams }) =>
        usePaginatedNavigation({
          data: { total: 50 },
          page: 1,
          totalPages: 5,
          setPage: vi.fn(),
          queryParams,
          prefetch,
        }),
      { initialProps: { queryParams: { page: 1, sortBy: "createdAt" } } },
    );

    expect(prefetch.mock.calls.length).toBeGreaterThan(0);

    act(() => {
      rerender({ queryParams: { page: 1, sortBy: "lastUsedAt" } });
    });

    const newSortCalls = prefetch.mock.calls.filter((call) => call[0].sortBy === "lastUsedAt");
    expect(newSortCalls.map((call) => call[0].page).sort((a, b) => a - b)).toEqual([2, 3]);
  });

  it("re-prefetches when prefetchKey changes while page and queryParams hold steady", () => {
    // Callers whose prefetch closure captures a value absent from queryParams
    // (api-keys' keyAuthId) pass it as prefetchKey. Switching keyspaces leaves
    // page, totalPages and queryParams identity untouched, so without that key
    // the adjacent pages of the new keyspace are never warmed.
    const prefetch = vi.fn();
    const queryParams = { page: 1 };
    const { rerender } = renderHook(
      ({ prefetchKey }) =>
        usePaginatedNavigation({
          data: { total: 50 },
          page: 1,
          totalPages: 5,
          setPage: vi.fn(),
          queryParams,
          prefetch,
          prefetchKey,
        }),
      { initialProps: { prefetchKey: "ks_a" } },
    );

    prefetch.mockClear();

    act(() => {
      rerender({ prefetchKey: "ks_b" });
    });

    expect(prefetch.mock.calls.map((call) => call[0].page).sort((a, b) => a - b)).toEqual([2, 3]);
  });

  it("suspends the clamp and the prefetch when disabled, but still guards onPageChange", () => {
    // Live-tail views (sentinel, ratelimit logs) pin themselves to page 1 and
    // hide the footer; clamping or warming pages there is wasted work.
    const prefetch = vi.fn();
    const setPage = vi.fn();
    const { result } = renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 9,
        totalPages: 2,
        setPage,
        queryParams: { page: 9 },
        prefetch,
        enabled: false,
      }),
    );

    expect(prefetch).not.toHaveBeenCalled();
    expect(setPage).not.toHaveBeenCalled();

    act(() => {
      result.current.onPageChange(5);
    });
    expect(setPage).not.toHaveBeenCalled();

    act(() => {
      result.current.onPageChange(2);
    });
    expect(setPage).toHaveBeenCalledWith(2);
  });

  it("onPageChange ignores out-of-range targets and navigates in-range ones", () => {
    const setPage = vi.fn();
    const { result } = renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 1,
        totalPages: 2,
        setPage,
        queryParams: { page: 1 },
        prefetch: vi.fn(),
      }),
    );

    act(() => {
      result.current.onPageChange(5);
    });
    act(() => {
      result.current.onPageChange(0);
    });
    expect(setPage).not.toHaveBeenCalled();

    act(() => {
      result.current.onPageChange(2);
    });
    expect(setPage).toHaveBeenCalledTimes(1);
    expect(setPage).toHaveBeenCalledWith(2);
  });
});

// normalizePageSize guards the divisor behind computeTotalPages and the `limit`
// sent to the server, so a page size of 0 escaping the clamp turns totalPages
// into Infinity and makes the endpoint reject the query.
describe("normalizePageSize", () => {
  it("clamps a fractional page size up to 1 rather than flooring it to 0", () => {
    expect(normalizePageSize(0.5, 50, 200)).toBe(1);
  });

  it("floors a fractional page size above 1", () => {
    expect(normalizePageSize(10.9, 50, 200)).toBe(10);
  });

  it("caps at maxPageSize and passes an in-range size through", () => {
    expect(normalizePageSize(500, 50, 200)).toBe(200);
    expect(normalizePageSize(25, 50, 200)).toBe(25);
  });

  it("falls back to the default for non-finite or non-positive input", () => {
    expect(normalizePageSize(Number.NaN, 50, 200)).toBe(50);
    expect(normalizePageSize(Number.POSITIVE_INFINITY, 50, 200)).toBe(50);
    expect(normalizePageSize(0, 50, 200)).toBe(50);
    expect(normalizePageSize(-10, 50, 200)).toBe(50);
  });
});

// computeFallbackTotalPages drives the clamp when the server cannot return a
// total count. The guarantees: Next reaches exactly one page past proven data,
// an overshoot snaps back to the last real page (page 1 for a stale deep
// link), and an in-flight fetch never re-triggers the collapse under
// keepPreviousData, which would cascade the clamp toward page 1.
describe("computeFallbackTotalPages", () => {
  const base = { isFetching: false, hasData: true, limit: 50, lastNonEmptyPage: 1 };

  it("advertises one page ahead only while the current page is full", () => {
    expect(computeFallbackTotalPages({ ...base, pageRowCount: 50, queryPage: 2 })).toBe(3);
    expect(computeFallbackTotalPages({ ...base, pageRowCount: 20, queryPage: 2 })).toBe(2);
  });

  it("snaps a settled empty page back to the last page seen with rows", () => {
    // Boundary-aligned data: pages 1-3 full, Next reached the empty page 4.
    expect(
      computeFallbackTotalPages({ ...base, pageRowCount: 0, queryPage: 4, lastNonEmptyPage: 3 }),
    ).toBe(3);
  });

  it("never reports a total at or above the current empty page", () => {
    // A racy lastNonEmptyPage must not leave the user stranded on the empty page.
    expect(
      computeFallbackTotalPages({ ...base, pageRowCount: 0, queryPage: 4, lastNonEmptyPage: 9 }),
    ).toBe(3);
  });

  it("snaps a stale deep link that never saw data back to page 1", () => {
    expect(computeFallbackTotalPages({ ...base, pageRowCount: 0, queryPage: 999 })).toBe(1);
  });

  it("holds the current page while its fetch is in flight", () => {
    // keepPreviousData shows the previous page's empty result during the
    // fetch; collapsing here would cascade the clamp 4 -> 3 -> 2 -> 1.
    expect(
      computeFallbackTotalPages({
        ...base,
        isFetching: true,
        pageRowCount: 0,
        queryPage: 3,
        lastNonEmptyPage: 3,
      }),
    ).toBe(3);
  });

  it("keeps a pre-data render at the current page so the clamp stays quiet", () => {
    expect(
      computeFallbackTotalPages({
        ...base,
        isFetching: true,
        hasData: false,
        pageRowCount: 0,
        queryPage: 5,
      }),
    ).toBe(5);
  });

  it("treats an empty first page as a single page", () => {
    expect(computeFallbackTotalPages({ ...base, pageRowCount: 0, queryPage: 1 })).toBe(1);
  });
});

// paginationFilterKey feeds the reset key the page hooks compare across
// renders. The load-bearing guarantee is content-stability: equal filter
// content must yield equal keys, and any meaningful difference must yield a
// different key. A regression here would silently break (or spuriously
// trigger) page resets across every table.
describe("paginationFilterKey", () => {
  it("produces the same key for equal content in a fresh array", () => {
    const a = paginationFilterKey([{ field: "status", operator: "is", value: "blocked" }]);
    const b = paginationFilterKey([{ field: "status", operator: "is", value: "blocked" }]);
    expect(a).toBe(b);
  });

  it("produces a different key when any component changes", () => {
    const base = paginationFilterKey([{ field: "status", operator: "is", value: "blocked" }]);
    expect(paginationFilterKey([{ field: "status", operator: "is", value: "passed" }])).not.toBe(
      base,
    );
    expect(paginationFilterKey([{ field: "status", operator: "not", value: "blocked" }])).not.toBe(
      base,
    );
    expect(paginationFilterKey([{ field: "outcome", operator: "is", value: "blocked" }])).not.toBe(
      base,
    );
  });

  // The reason this uses JSON tuples instead of `field:op:value` joined by `|`:
  // a value containing the delimiter must not let two distinct states collapse
  // to the same key, which would suppress a real page reset.
  it("does not collide when a value contains delimiter characters", () => {
    const oneFilter = paginationFilterKey([{ field: "name", operator: "is", value: "a|b" }]);
    const twoFilters = paginationFilterKey([
      { field: "name", operator: "is", value: "a" },
      { field: "b", operator: "is", value: "" },
    ]);
    expect(oneFilter).not.toBe(twoFilters);

    const colonValue = paginationFilterKey([{ field: "name", operator: "is", value: "a:b" }]);
    const colonSplit = paginationFilterKey([{ field: "name", operator: "is:a", value: "b" }]);
    expect(colonValue).not.toBe(colonSplit);
  });

  // Filter values are not always strings (timestamps, counts); encoding must be
  // deterministic so the key stays stable for equal values.
  it("encodes non-string values deterministically", () => {
    const a = paginationFilterKey([{ field: "startTime", operator: "is", value: 1700000000 }]);
    const b = paginationFilterKey([{ field: "startTime", operator: "is", value: 1700000000 }]);
    expect(a).toBe(b);
    // A number and its string form are distinct states and must differ.
    expect(
      paginationFilterKey([{ field: "startTime", operator: "is", value: "1700000000" }]),
    ).not.toBe(a);
  });

  it("returns a stable key for no filters", () => {
    expect(paginationFilterKey([])).toBe(paginationFilterKey([]));
  });
});

// paginationSortKey carries the same guarantee as paginationFilterKey for the
// multi-column `useSort` surface: equal sort content yields equal keys, and
// any difference in order, column, or direction yields a different key.
describe("paginationSortKey", () => {
  it("produces the same key for equal content in a fresh array", () => {
    const a = paginationSortKey([{ column: "time", direction: "desc" }]);
    const b = paginationSortKey([{ column: "time", direction: "desc" }]);
    expect(a).toBe(b);
  });

  it("produces a different key when order, column, or direction changes", () => {
    const base = paginationSortKey([
      { column: "time", direction: "desc" },
      { column: "name", direction: "asc" },
    ]);
    expect(
      paginationSortKey([
        { column: "name", direction: "asc" },
        { column: "time", direction: "desc" },
      ]),
    ).not.toBe(base);
    expect(paginationSortKey([{ column: "time", direction: "asc" }])).not.toBe(
      paginationSortKey([{ column: "time", direction: "desc" }]),
    );
  });

  it("returns a stable key for no sorts", () => {
    expect(paginationSortKey([])).toBe(paginationSortKey([]));
  });
});
