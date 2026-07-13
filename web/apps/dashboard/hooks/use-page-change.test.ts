import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePageChange } from "./use-page-change";

describe("usePageChange", () => {
  let setPage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    setPage = vi.fn();
  });

  it("sets an in-range page", () => {
    const { result } = renderHook(() => usePageChange(10, setPage));

    result.current(4);

    expect(setPage).toHaveBeenCalledWith(4);
  });

  // A stale pager control must not be able to move the user off the valid range.
  it("ignores out-of-range requests", () => {
    const { result } = renderHook(() => usePageChange(10, setPage));

    result.current(0);
    result.current(-1);
    result.current(11);

    expect(setPage).not.toHaveBeenCalled();
  });

  it("treats the boundaries as in range", () => {
    const { result } = renderHook(() => usePageChange(10, setPage));

    result.current(1);
    result.current(10);

    expect(setPage).toHaveBeenNthCalledWith(1, 1);
    expect(setPage).toHaveBeenNthCalledWith(2, 10);
  });

  // totalPages=1 (the only-one-page and pre-data-collapse case) still accepts
  // page 1 and rejects everything else.
  it("allows only page 1 when there is a single page", () => {
    const { result } = renderHook(() => usePageChange(1, setPage));

    result.current(1);
    result.current(2);

    expect(setPage).toHaveBeenCalledTimes(1);
    expect(setPage).toHaveBeenCalledWith(1);
  });
});
