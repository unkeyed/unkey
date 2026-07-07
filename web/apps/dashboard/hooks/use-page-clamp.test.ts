import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePageClamp } from "./use-page-clamp";

describe("usePageClamp", () => {
  let setPage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    setPage = vi.fn();
  });

  it("clamps the page down once the query has settled with fewer pages", () => {
    renderHook(() =>
      usePageClamp({ page: 7, totalPages: 3, isFetching: false, hasData: true, setPage }),
    );

    expect(setPage).toHaveBeenCalledWith(3);
  });

  it("leaves an in-range page untouched", () => {
    renderHook(() =>
      usePageClamp({ page: 2, totalPages: 3, isFetching: false, hasData: true, setPage }),
    );

    expect(setPage).not.toHaveBeenCalled();
  });

  // Guards the ENG-2935 regression: with keepPreviousData, totalPages reflects
  // the previous query while a filter/sort/time change is in flight. Clamping
  // against that stale total would race the reset-to-page-1 effect.
  it("does not clamp against stale totals while a fetch is in flight", () => {
    const { rerender } = renderHook((props) => usePageClamp(props), {
      initialProps: { page: 7, totalPages: 3, isFetching: true, hasData: true, setPage },
    });

    expect(setPage).not.toHaveBeenCalled();

    // The fetch settles with the real total for the new params — only now
    // may the clamp run.
    rerender({ page: 7, totalPages: 5, isFetching: false, hasData: true, setPage });

    expect(setPage).toHaveBeenCalledTimes(1);
    expect(setPage).toHaveBeenCalledWith(5);
  });

  // Guards deep links: before the first response, totalPages collapses to 1,
  // and clamping then would snap a URL-persisted ?page=7 back to page 1.
  it("does not clamp a deep-linked page before the first result loads", () => {
    const { rerender } = renderHook((props) => usePageClamp(props), {
      initialProps: { page: 7, totalPages: 1, isFetching: false, hasData: false, setPage },
    });

    expect(setPage).not.toHaveBeenCalled();

    rerender({ page: 7, totalPages: 10, isFetching: false, hasData: true, setPage });

    expect(setPage).not.toHaveBeenCalled();
  });
});
