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
  computeTotalPages,
  normalizePageSize,
  paginationFilterKey,
  paginationSortKey,
  useFallbackTotalPages,
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

// usePaginatedNavigation requires the raw query flags so no caller can silently
// lose its loading states. Tests that exercise the clamp/prefetch rather than
// those states pass the settled combination.
const SETTLED = { isLoading: false, isFetching: false } as const;

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
  describe("deep-link clamp guarantee", () => {
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

    it("collapses a multi-sort URL to the first valid entry", () => {
      // The server takes one sortBy/sortOrder, so `sorting` must not advertise
      // the trailing entries: header indicators (and callers that re-sort rows
      // off `sorting`) would claim an ordering the rows do not have.
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      urlStore.sort = [
        { column: "createdAt", direction: "asc" },
        { column: "updatedAt", direction: "desc" },
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

      expect(result.current.sorting).toEqual([{ id: "created", desc: false }]);
    });

    it("skips unknown leading columns and sorts by the first known one", () => {
      queryResult = { data: { total: 100 }, isLoading: false, isFetching: false };
      urlStore.sort = [
        { column: "bogus", direction: "asc" },
        { column: "updatedAt", direction: "desc" },
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

      expect(result.current.sorting).toEqual([{ id: "updated", desc: true }]);
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
  // Without a synchronous reset, the render that observes a filter change would
  // feed the stale page into the clamp while keepPreviousData still reports the
  // previous result set's `totalPages`, snapping the page to the old last page.
  // usePaginatedPage surfaces page 1 on that same render, closing the window.
  // This test pins that guarantee: lose the synchronous reset and it fails.
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
          ...SETTLED,
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
        ...SETTLED,
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
        ...SETTLED,
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
        ...SETTLED,
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
        ...SETTLED,
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
          ...SETTLED,
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
          ...SETTLED,
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

  it("does not re-run the prefetch when only the prefetch identity changes", () => {
    // Every caller passes a fresh inline arrow each render
    // (`prefetch: (params) => utils.x.prefetch(...)`), so the ref indirection is
    // what keeps the effect from firing on every render of every list view.
    // Without it each render would issue PREFETCH_PAGES_AHEAD tRPC requests.
    const first = vi.fn();
    const next = vi.fn();
    const queryParams = { page: 1 };
    const { rerender } = renderHook(
      ({ prefetch }) =>
        usePaginatedNavigation({
          data: { total: 50 },
          page: 1,
          totalPages: 5,
          setPage: vi.fn(),
          ...SETTLED,
          queryParams,
          prefetch,
        }),
      { initialProps: { prefetch: first } },
    );

    expect(first).toHaveBeenCalledTimes(2);
    first.mockClear();

    rerender({ prefetch: next });

    expect(next).not.toHaveBeenCalled();
    expect(first).not.toHaveBeenCalled();
  });

  it("suspends only the clamp when clampEnabled is false, but still guards onPageChange", () => {
    // Live-tail views (sentinel, ratelimit logs) pin themselves to page 1 and
    // hide the footer, so clamping the pinned page would fight the pin. The
    // prefetch keeps running so pages are warm when the user leaves live tail.
    const prefetch = vi.fn();
    const setPage = vi.fn();
    const { result } = renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 1,
        totalPages: 4,
        setPage,
        ...SETTLED,
        queryParams: { page: 1 },
        prefetch,
        clampEnabled: false,
      }),
    );

    expect(prefetch.mock.calls.map((call) => call[0].page).sort((a, b) => a - b)).toEqual([2, 3]);
    expect(setPage).not.toHaveBeenCalled();

    act(() => {
      result.current.onPageChange(9);
    });
    expect(setPage).not.toHaveBeenCalled();

    act(() => {
      result.current.onPageChange(2);
    });
    expect(setPage).toHaveBeenCalledWith(2);
  });

  it("does not clamp an out-of-range page while clampEnabled is false", () => {
    const setPage = vi.fn();
    renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 9,
        totalPages: 2,
        setPage,
        ...SETTLED,
        queryParams: { page: 9 },
        prefetch: vi.fn(),
        clampEnabled: false,
      }),
    );

    expect(setPage).not.toHaveBeenCalled();
  });

  it("separates initial load from page-to-page navigation", () => {
    // keepPreviousData keeps the previous rows on screen during a page change,
    // so isFetching alone cannot tell a first load from a navigation.
    const { result: initial } = renderHook(() =>
      usePaginatedNavigation({
        data: undefined,
        page: 1,
        totalPages: 1,
        setPage: vi.fn(),
        isLoading: true,
        isFetching: true,
        queryParams: { page: 1 },
        prefetch: vi.fn(),
      }),
    );
    expect(initial.current.isInitialLoading).toBe(true);
    expect(initial.current.isNavigating).toBe(false);

    const { result: navigating } = renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 2,
        totalPages: 2,
        setPage: vi.fn(),
        isLoading: false,
        isFetching: true,
        queryParams: { page: 2 },
        prefetch: vi.fn(),
      }),
    );
    expect(navigating.current.isInitialLoading).toBe(false);
    expect(navigating.current.isNavigating).toBe(true);
  });

  it("onPageChange ignores out-of-range targets and navigates in-range ones", () => {
    const setPage = vi.fn();
    const { result } = renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 1,
        totalPages: 2,
        setPage,
        ...SETTLED,
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

  it("does not advertise a page ahead off the previous page's rows mid-fetch", () => {
    // On page 4 with page 3's full result still on screen (keepPreviousData).
    // Crediting those 50 rows to page 4 would report totalPages 5 and let a
    // second Next click jump to page 5, skipping page 4 entirely — and since
    // page 4 is then never observed, lastNonEmptyPage stays 3 and the empty
    // branch above later clamps back to 3, past a page that did have rows.
    expect(
      computeFallbackTotalPages({
        ...base,
        isFetching: true,
        pageRowCount: 50,
        queryPage: 4,
        lastNonEmptyPage: 3,
      }),
    ).toBe(4);
  });

  it("keeps a proven page reachable while an overshoot is in flight", () => {
    // Mid-fetch on page 9 after page 3 was proven: the total must not collapse
    // below the last proven page, or the clamp would fire before the response
    // lands and strand the user behind data that exists.
    expect(
      computeFallbackTotalPages({
        ...base,
        isFetching: true,
        pageRowCount: 0,
        queryPage: 1,
        lastNonEmptyPage: 3,
      }),
    ).toBe(3);
  });
});

// useFallbackTotalPages wraps computeFallbackTotalPages with the observed-page
// memory the caller used to own. The invariants: a page is only remembered once
// its fetch settles with rows, and the memory is dropped when the reset key
// changes, since rows seen under the old filters prove nothing about the new.
describe("useFallbackTotalPages", () => {
  const base = { hasData: true, limit: 50, resetKey: "filters-a" };

  it("remembers a settled non-empty page and snaps a later empty page back to it", () => {
    const { result, rerender } = renderHook(
      (props: Parameters<typeof useFallbackTotalPages>[0]) => useFallbackTotalPages(props),
      { initialProps: { ...base, isFetching: true, pageRowCount: 0, queryPage: 1 } },
    );

    // Page 3 settles with a full result: reachable one page ahead.
    rerender({ ...base, isFetching: false, pageRowCount: 50, queryPage: 3 });
    expect(result.current).toBe(4);

    // Page 4 comes back empty: snap back to the last page seen with rows.
    rerender({ ...base, isFetching: false, pageRowCount: 0, queryPage: 4 });
    expect(result.current).toBe(3);
  });

  it("does not remember a page whose fetch has not settled", () => {
    const { result, rerender } = renderHook(
      (props: Parameters<typeof useFallbackTotalPages>[0]) => useFallbackTotalPages(props),
      { initialProps: { ...base, isFetching: true, pageRowCount: 0, queryPage: 1 } },
    );

    // Rows on screen belong to the previous page, so page 4 is not proven.
    rerender({ ...base, isFetching: true, pageRowCount: 50, queryPage: 4 });
    expect(result.current).toBe(4);

    // Page 4 settles empty: nothing was ever proven, so collapse to page 1.
    rerender({ ...base, isFetching: false, pageRowCount: 0, queryPage: 4 });
    expect(result.current).toBe(1);
  });

  it("drops the remembered page when the reset key changes", () => {
    const { result, rerender } = renderHook(
      (props: Parameters<typeof useFallbackTotalPages>[0]) => useFallbackTotalPages(props),
      { initialProps: { ...base, isFetching: true, pageRowCount: 0, queryPage: 1 } },
    );

    rerender({ ...base, isFetching: false, pageRowCount: 50, queryPage: 3 });
    expect(result.current).toBe(4);

    // New filters: usePaginatedPage has already snapped the page to 1, and the
    // old page-3 memory must not survive into the new result set.
    rerender({
      ...base,
      resetKey: "filters-b",
      isFetching: true,
      pageRowCount: 0,
      queryPage: 1,
    });
    expect(result.current).toBe(1);
  });
});

// computeTotalPages is what every migrated hook routes its footer through.
// The floor is load-bearing: a 0 here would make the clamp call setPage(0).
describe("computeTotalPages", () => {
  it("rounds a partial last page up", () => {
    expect(computeTotalPages(101, 50)).toBe(3);
    expect(computeTotalPages(100, 50)).toBe(2);
  });

  it("never reports fewer than one page for an empty list", () => {
    expect(computeTotalPages(0, 50)).toBe(1);
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
