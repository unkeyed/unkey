import { act, renderHook } from "@testing-library/react";
import { useCallback, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// In-memory stand-in for the URL query string. Only `useQueryState` is swapped;
// the rest of nuqs stays real. Mirrors the fake in use-paginated-list-query.test.tsx.
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

import { usePaginatedNavigation, usePaginatedPage } from "./use-paginated-list-query";

beforeEach(() => {
  for (const key of Object.keys(urlStore)) {
    delete urlStore[key];
  }
});

afterEach(() => {
  vi.clearAllMocks();
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
        prefetch: vi.fn(),
      }),
    );

    expect(setPage).toHaveBeenCalledWith(2);
  });

  it("prefetches the pages ahead and stops at the last page", () => {
    const prefetch = vi.fn();
    renderHook(() =>
      usePaginatedNavigation({
        data: { total: 50 },
        page: 1,
        totalPages: 5,
        setPage: vi.fn(),
        prefetch,
      }),
    );

    const prefetched = [...new Set(prefetch.mock.calls.map((call) => call[0]))].sort(
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
        prefetch,
      }),
    );

    expect(prefetch).not.toHaveBeenCalled();
  });

  it("onPageChange ignores out-of-range targets and navigates in-range ones", () => {
    const setPage = vi.fn();
    const { result } = renderHook(() =>
      usePaginatedNavigation({
        data: { total: 20 },
        page: 1,
        totalPages: 2,
        setPage,
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
