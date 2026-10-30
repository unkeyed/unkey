import { describe, expect, it } from "vitest";
import { type Grid, type Held, clampedRetryMs, resolveHeld } from "./app-metrics";

const asked: Grid = {
  range: "1h",
  bucket: "1m",
  bucketMs: 60_000,
  startMs: 0,
  endMs: 3_600_000,
};

function served(endMs: number): Grid {
  return { ...asked, startMs: endMs - 3_600_000, endMs };
}

describe("clampedRetryMs", () => {
  it("retries a result clamped to the previous bucket", () => {
    expect(clampedRetryMs(asked, served(3_540_000), 1)).toBe(2_000);
    expect(clampedRetryMs(asked, served(3_540_000), 2)).toBe(2_000);
  });

  it("stops after two retries so a fast browser clock waits for the next tick", () => {
    expect(clampedRetryMs(asked, served(3_540_000), 3)).toBe(false);
    expect(clampedRetryMs(asked, served(3_540_000), 40)).toBe(false);
  });

  it("does not retry a result that matches the asked grid", () => {
    expect(clampedRetryMs(asked, served(3_600_000), 1)).toBe(false);
  });

  it("does not retry a gap wider than one bucket", () => {
    expect(clampedRetryMs(asked, served(3_480_000), 1)).toBe(false);
  });
});

describe("resolveHeld", () => {
  const held: Held<string[], string> = {
    viewKey: "app:prod",
    range: "1h",
    data: ["old"],
    failedRange: null,
  };
  const view = {
    viewKey: "app:prod",
    range: "6h",
    hold: "across-ranges" as const,
    fresh: undefined,
    isError: false,
  };

  it("uses fresh data, holds it, and clears a latch", () => {
    expect(
      resolveHeld({ ...held, failedRange: "6h" }, { ...view, fresh: ["new"], isError: true }),
    ).toEqual({
      data: ["new"],
      held: { viewKey: "app:prod", range: "6h", data: ["new"], failedRange: null },
    });
  });

  it("keeps the same held state for fresh data already held", () => {
    const current = { ...held, range: "6h" };
    expect(resolveHeld(current, { ...view, fresh: current.data }).held).toBe(current);
  });

  it("clears a latch when fresh data is already held", () => {
    const latched = { ...held, failedRange: "6h" };
    expect(resolveHeld(latched, { ...view, range: "1h", fresh: latched.data })).toEqual({
      data: ["old"],
      held: { viewKey: "app:prod", range: "1h", data: ["old"], failedRange: null },
    });
  });

  it("uses held data for the same view and range, even on error", () => {
    const result = resolveHeld(held, { ...view, range: "1h", isError: true });
    expect(result).toEqual({ data: ["old"], held });
    expect(result.held).toBe(held);
  });

  it("clears a latch on return to the held range, then keeps it stable", () => {
    const first = resolveHeld({ ...held, failedRange: "6h" }, { ...view, range: "1h" });
    expect(first).toEqual({
      data: ["old"],
      held: { viewKey: "app:prod", range: "1h", data: ["old"], failedRange: null },
    });
    expect(resolveHeld(first.held, { ...view, range: "1h" }).held).toBe(first.held);
  });

  it("uses the old range's data on a new range holding across ranges", () => {
    const result = resolveHeld(held, view);
    expect(result).toEqual({ data: ["old"], held });
    expect(result.held).toBe(held);
  });

  it("uses held data on a range other than the latched one", () => {
    const latched = { ...held, failedRange: "6h" };
    const result = resolveHeld(latched, { ...view, range: "24h" });
    expect(result).toEqual({ data: ["old"], held: latched });
    expect(result.held).toBe(latched);
  });

  it("latches a new range that errors as failed with no data", () => {
    expect(resolveHeld(held, { ...view, isError: true })).toEqual({
      data: undefined,
      held: { viewKey: "app:prod", range: "1h", data: ["old"], failedRange: "6h" },
    });
  });

  it("shows no data for a latched range pending again", () => {
    const latched = { ...held, failedRange: "6h" };
    const result = resolveHeld(latched, view);
    expect(result).toEqual({ data: undefined, held: latched });
    expect(result.held).toBe(latched);
  });

  it("keeps the same latch for a latched range that errors again", () => {
    const latched = { ...held, failedRange: "6h" };
    const result = resolveHeld(latched, { ...view, isError: true });
    expect(result).toEqual({ data: undefined, held: latched });
    expect(result.held).toBe(latched);
  });

  it("shows no data on a new range when holding within a range", () => {
    const result = resolveHeld(held, { ...view, hold: "within-range" });
    expect(result).toEqual({ data: undefined, held });
    expect(result.held).toBe(held);
  });

  it("does not latch an error when holding within a range", () => {
    const result = resolveHeld(held, { ...view, hold: "within-range", isError: true });
    expect(result).toEqual({ data: undefined, held });
    expect(result.held).toBe(held);
  });

  it("shows no data for a different view", () => {
    const result = resolveHeld(held, { ...view, viewKey: "app:staging", range: "1h" });
    expect(result).toEqual({ data: undefined, held });
    expect(result.held).toBe(held);
  });
});
