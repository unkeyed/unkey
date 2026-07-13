import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePrefetchPages } from "./use-prefetch-pages";

type Params = { page: number; filter: string };

describe("usePrefetchPages", () => {
  let prefetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    prefetch = vi.fn();
  });

  it("prefetches the next `pagesAhead` pages, overriding only `page`", () => {
    renderHook(() =>
      usePrefetchPages<Params>({
        page: 2,
        totalPages: 10,
        queryParams: { page: 2, filter: "a" },
        prefetch,
      }),
    );

    expect(prefetch).toHaveBeenCalledTimes(2);
    expect(prefetch).toHaveBeenNthCalledWith(1, { page: 3, filter: "a" });
    expect(prefetch).toHaveBeenNthCalledWith(2, { page: 4, filter: "a" });
  });

  it("stops at totalPages and never prefetches a page that does not exist", () => {
    renderHook(() =>
      usePrefetchPages<Params>({
        page: 10,
        totalPages: 10,
        queryParams: { page: 10, filter: "a" },
        prefetch,
      }),
    );

    expect(prefetch).not.toHaveBeenCalled();
  });

  it("honors a custom pagesAhead", () => {
    renderHook(() =>
      usePrefetchPages<Params>({
        page: 1,
        totalPages: 10,
        queryParams: { page: 1, filter: "a" },
        prefetch,
        pagesAhead: 1,
      }),
    );

    expect(prefetch).toHaveBeenCalledTimes(1);
    expect(prefetch).toHaveBeenCalledWith({ page: 2, filter: "a" });
  });

  // Callers pass a fresh arrow (or tRPC proxy) each render; the ref must keep
  // the effect from re-firing when only the callback identity changes. Real
  // callers memoize queryParams, so it is held stable here to isolate the
  // callback-identity concern.
  it("does not re-run when only the prefetch identity changes", () => {
    const queryParams: Params = { page: 1, filter: "a" };
    const { rerender } = renderHook(
      (props: { prefetch: (p: Params) => void }) =>
        usePrefetchPages<Params>({
          page: 1,
          totalPages: 10,
          queryParams,
          prefetch: props.prefetch,
        }),
      { initialProps: { prefetch } },
    );

    expect(prefetch).toHaveBeenCalledTimes(2);

    // Same inputs, brand-new callback identity — effect must not re-fire.
    const next = vi.fn();
    rerender({ prefetch: next });

    expect(prefetch).toHaveBeenCalledTimes(2);
    expect(next).not.toHaveBeenCalled();
  });

  it("re-runs when the page changes", () => {
    const { rerender } = renderHook(
      (props: { page: number }) =>
        usePrefetchPages<Params>({
          page: props.page,
          totalPages: 10,
          queryParams: { page: props.page, filter: "a" },
          prefetch,
        }),
      { initialProps: { page: 1 } },
    );

    prefetch.mockClear();
    rerender({ page: 5 });

    expect(prefetch).toHaveBeenCalledTimes(2);
    expect(prefetch).toHaveBeenNthCalledWith(1, { page: 6, filter: "a" });
    expect(prefetch).toHaveBeenNthCalledWith(2, { page: 7, filter: "a" });
  });
});
