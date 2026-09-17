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
    const { result } = renderHook(() => useNow(10_000));
    const mounted = result.current;

    act(() => {
      vi.advanceTimersByTime(30_000);
    });

    expect(result.current - mounted).toBe(30_000);
  });

  it("stops reading the clock once unmounted", () => {
    let renders = 0;
    const { unmount } = renderHook(() => {
      renders += 1;
      return useNow(10_000);
    });
    const before = renders;

    unmount();
    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    expect(renders).toBe(before);
  });
});
