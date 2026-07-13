import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePageClamp } from "./use-page-clamp";
import { usePageTransition } from "./use-page-transition";

describe("usePageClamp", () => {
  let setPage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    setPage = vi.fn();
  });

  it("clamps the page down once a result with fewer pages is available", () => {
    renderHook(() => usePageClamp({ page: 7, totalPages: 3, data: { total: 150 }, setPage }));

    expect(setPage).toHaveBeenCalledWith(3);
  });

  it("leaves an in-range page untouched", () => {
    renderHook(() => usePageClamp({ page: 2, totalPages: 3, data: { total: 150 }, setPage }));

    expect(setPage).not.toHaveBeenCalled();
  });

  // Guards deep links: before the first response, totalPages collapses to 1,
  // and clamping then would snap a URL-persisted ?page=7 back to page 1.
  it("does not clamp a deep-linked page before the first result loads", () => {
    const { rerender } = renderHook((props) => usePageClamp(props), {
      initialProps: { page: 7, totalPages: 1, data: undefined as unknown, setPage },
    });

    expect(setPage).not.toHaveBeenCalled();

    rerender({ page: 7, totalPages: 10, data: { total: 500 }, setPage });

    expect(setPage).not.toHaveBeenCalled();
  });
});

// Guards the ENG-2935 regression end to end: with keepPreviousData and
// staleTime: Infinity, a filter change can resolve synchronously from the
// cache, so no fetch state ever flips. The clamp must compare the
// transition-aware page (1) — not the outgoing URL page — or its
// setPage(totalPages) races the reset-to-page-1 and lands the table on a
// wrong page of the new result set.
describe("usePageTransition + usePageClamp", () => {
  it("lands on page 1, not a clamped page, when a filter change resolves from cache", () => {
    const setPage = vi.fn();

    const useHarness = (props: {
      transitionKey: string;
      page: number;
      totalPages: number;
      data: unknown;
    }) => {
      const queryPage = usePageTransition({
        transitionKey: props.transitionKey,
        page: props.page,
        setPage,
      });
      usePageClamp({ page: queryPage, totalPages: props.totalPages, data: props.data, setPage });
      return queryPage;
    };

    const { result, rerender } = renderHook(useHarness, {
      initialProps: { transitionKey: "filter:A", page: 7, totalPages: 10, data: { total: 500 } },
    });

    // Cache hit: data for filter B — only 3 pages, fewer than the outgoing
    // page 7 — arrives in the same render as the key change.
    rerender({ transitionKey: "filter:B", page: 7, totalPages: 3, data: { total: 150 } });

    expect(result.current).toBe(1);
    expect(setPage).toHaveBeenCalledWith(1);
    expect(setPage).not.toHaveBeenCalledWith(3);
  });
});
