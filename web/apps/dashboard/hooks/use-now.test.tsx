import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useNow } from "./use-now";

describe("useNow", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("advances on its own, without a re-render from the outside", () => {
    const { result } = renderHook(() => useNow(30_000));
    const mounted = result.current;

    act(() => {
      vi.advanceTimersByTime(90_000);
    });

    expect(result.current - mounted).toBe(90_000);
  });

  it("reads the clock again on remount", () => {
    const first = renderHook(() => useNow(30_000));
    first.unmount();

    act(() => {
      vi.advanceTimersByTime(600_000);
    });

    const second = renderHook(() => useNow(30_000));
    expect(second.result.current).toBe(Date.now());
  });

  it("stops ticking once unmounted", () => {
    const clear = vi.spyOn(globalThis, "clearInterval");
    renderHook(() => useNow(30_000)).unmount();
    expect(clear).toHaveBeenCalled();
  });
});
