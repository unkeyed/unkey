import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePageTransition } from "./use-page-transition";

describe("usePageTransition", () => {
  let setPage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    setPage = vi.fn();
  });

  // Guards deep links: mounting with ?page=7 must not count as a transition.
  it("preserves a URL-persisted page on first mount", () => {
    const { result } = renderHook(() =>
      usePageTransition({ transitionKey: "filter:A", page: 7, setPage }),
    );

    expect(result.current).toBe(7);
    expect(setPage).not.toHaveBeenCalled();
  });

  // The query must never fire with the outgoing page against the new key —
  // that request would 404-page semantically and, with staleTime: Infinity,
  // poison the cache forever.
  it("returns page 1 synchronously on the render that observes a key change", () => {
    const { result, rerender } = renderHook((props) => usePageTransition(props), {
      initialProps: { transitionKey: "filter:A", page: 7, setPage },
    });

    rerender({ transitionKey: "filter:B", page: 7, setPage });

    expect(result.current).toBe(1);
    expect(setPage).toHaveBeenCalledTimes(1);
    expect(setPage).toHaveBeenCalledWith(1);
  });

  it("returns the live page again once the reset has committed", () => {
    const { result, rerender } = renderHook((props) => usePageTransition(props), {
      initialProps: { transitionKey: "filter:A", page: 7, setPage },
    });

    rerender({ transitionKey: "filter:B", page: 7, setPage });
    // The setPage(1) effect has run; the URL now reflects page 1.
    rerender({ transitionKey: "filter:B", page: 1, setPage });

    expect(result.current).toBe(1);
    expect(setPage).toHaveBeenCalledTimes(1);

    // Navigation within the new key is passed through untouched.
    rerender({ transitionKey: "filter:B", page: 4, setPage });
    expect(result.current).toBe(4);
    expect(setPage).toHaveBeenCalledTimes(1);
  });

  it("runs onTransition alongside the page reset, but not on mount", () => {
    const onTransition = vi.fn();
    const { rerender } = renderHook((props) => usePageTransition(props), {
      initialProps: { transitionKey: "filter:A", page: 7, setPage, onTransition },
    });

    expect(onTransition).not.toHaveBeenCalled();

    rerender({ transitionKey: "filter:B", page: 7, setPage, onTransition });

    expect(onTransition).toHaveBeenCalledTimes(1);
  });
});
