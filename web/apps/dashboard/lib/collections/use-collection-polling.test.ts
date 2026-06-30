import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCollectionPolling } from "./use-collection-polling";

describe("useCollectionPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not poll while disabled", () => {
    const refetch = vi.fn();
    renderHook(() => useCollectionPolling(refetch, { intervalMs: 5000, enabled: false }));

    vi.advanceTimersByTime(20_000);
    expect(refetch).not.toHaveBeenCalled();
  });

  it("polls on the interval while enabled", () => {
    const refetch = vi.fn();
    renderHook(() => useCollectionPolling(refetch, { intervalMs: 5000, enabled: true }));

    vi.advanceTimersByTime(12_000);
    expect(refetch).toHaveBeenCalledTimes(2);
  });

  it("stops polling when disabled", () => {
    const refetch = vi.fn();
    const { rerender } = renderHook(
      ({ enabled }) => useCollectionPolling(refetch, { intervalMs: 5000, enabled }),
      { initialProps: { enabled: true } },
    );

    vi.advanceTimersByTime(5000);
    expect(refetch).toHaveBeenCalledTimes(1);

    rerender({ enabled: false });
    vi.advanceTimersByTime(20_000);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("clears the interval on unmount", () => {
    const refetch = vi.fn();
    const { unmount } = renderHook(() =>
      useCollectionPolling(refetch, { intervalMs: 5000, enabled: true }),
    );

    vi.advanceTimersByTime(5000);
    expect(refetch).toHaveBeenCalledTimes(1);

    unmount();
    vi.advanceTimersByTime(20_000);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("uses the latest callback without restarting the interval", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(
      ({ cb }) => useCollectionPolling(cb, { intervalMs: 5000, enabled: true }),
      { initialProps: { cb: first } },
    );

    vi.advanceTimersByTime(5000);
    expect(first).toHaveBeenCalledTimes(1);

    rerender({ cb: second });
    vi.advanceTimersByTime(5000);
    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
  });
});
