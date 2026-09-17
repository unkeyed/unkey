import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useNow } from "./use-now";

describe("useNow", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00.000Z"));
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("advances on its own, without a re-render from the outside", () => {
    const { result } = renderHook(() => useNow());
    const mounted = result.current;

    act(() => {
      vi.advanceTimersByTime(30_000);
    });

    expect(result.current - mounted).toBe(30_000);
  });

  it("reads the same clock however late a caller mounts", () => {
    const first = renderHook(() => useNow());

    act(() => {
      vi.advanceTimersByTime(900);
    });
    const second = renderHook(() => useNow());

    expect(second.result.current).toBe(first.result.current);
  });

  it("runs one timer for every caller and stops it with the last of them", () => {
    const first = renderHook(() => useNow());
    const second = renderHook(() => useNow());
    expect(vi.getTimerCount()).toBe(1);

    first.unmount();
    expect(vi.getTimerCount()).toBe(1);

    second.unmount();
    expect(vi.getTimerCount()).toBe(0);
  });
});
