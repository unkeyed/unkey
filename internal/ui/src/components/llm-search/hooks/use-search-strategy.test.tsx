import { act, renderHook } from "@testing-library/react-hooks";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useSearchStrategy } from "./use-search-strategy";

describe("useSearchStrategy", () => {
  // Mock timers for debounce/throttle testing
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const onSearchMock = vi.fn();

  it("should execute search immediately with executeSearch", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    act(() => {
      result.current.executeSearch("test query");
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);
    expect(onSearchMock).toHaveBeenCalledWith("test query");
  });

  it("should not execute search with empty query", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    act(() => {
      result.current.executeSearch("  ");
    });

    expect(onSearchMock).not.toHaveBeenCalled();
  });

  it("should debounce search calls with debouncedSearch", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    act(() => {
      result.current.debouncedSearch("test query");
    });

    expect(onSearchMock).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(499);
    });

    expect(onSearchMock).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(1);
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);
    expect(onSearchMock).toHaveBeenCalledWith("test query");
  });

  it("should cancel previous debounce if debouncedSearch is called again", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    act(() => {
      result.current.debouncedSearch("first query");
    });

    act(() => {
      vi.advanceTimersByTime(300);
    });

    act(() => {
      result.current.debouncedSearch("second query");
    });

    act(() => {
      vi.advanceTimersByTime(300);
    });

    expect(onSearchMock).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);
    expect(onSearchMock).toHaveBeenCalledWith("second query");
    expect(onSearchMock).not.toHaveBeenCalledWith("first query");
  });

  it("should use debounce for initial query with throttledSearch", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    act(() => {
      result.current.throttledSearch("initial query");
    });

    expect(onSearchMock).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);
    expect(onSearchMock).toHaveBeenCalledWith("initial query");
  });

  it("should throttle subsequent searches", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    // First search - should be debounced
    act(() => {
      result.current.throttledSearch("initial query");
      vi.advanceTimersByTime(500);
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);

    // Reset mock to track subsequent calls
    onSearchMock.mockReset();

    // Second search immediately after - should be throttled
    act(() => {
      result.current.throttledSearch("second query");
    });

    // Should not execute immediately due to throttling
    expect(onSearchMock).not.toHaveBeenCalled();

    // Advance time to just before throttle interval ends
    act(() => {
      vi.advanceTimersByTime(999);
    });

    expect(onSearchMock).not.toHaveBeenCalled();

    // Complete the throttle interval
    act(() => {
      vi.advanceTimersByTime(1);
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);
    expect(onSearchMock).toHaveBeenCalledWith("second query");
  });

  it("should clean up timers with clearDebounceTimer", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    act(() => {
      result.current.debouncedSearch("test query");
    });

    act(() => {
      result.current.clearDebounceTimer();
    });

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(onSearchMock).not.toHaveBeenCalled();
  });

  it("should reset search state with resetSearchState", () => {
    const { result } = renderHook(() => useSearchStrategy(onSearchMock, 500));

    // First search to set initial state
    act(() => {
      result.current.throttledSearch("initial query");
      vi.advanceTimersByTime(500);
    });

    onSearchMock.mockReset();

    // Reset search state
    act(() => {
      result.current.resetSearchState();
    });

    // Next search should be debounced again, not throttled
    act(() => {
      result.current.throttledSearch("new query after reset");
    });

    // Should not execute immediately (debounced, not throttled)
    expect(onSearchMock).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(onSearchMock).toHaveBeenCalledTimes(1);
    expect(onSearchMock).toHaveBeenCalledWith("new query after reset");
  });
});
