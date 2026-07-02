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
