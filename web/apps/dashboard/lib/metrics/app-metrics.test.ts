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
  const held: Held<string[]> = { viewKey: "app:prod", range: "1h", data: ["old"] };
  const view = { viewKey: "app:prod", range: "6h" as const, fresh: undefined, isError: false };

  it("uses fresh data and holds it", () => {
    expect(resolveHeld(held, { ...view, fresh: ["new"], isError: true })).toEqual({
      data: ["new"],
      held: { viewKey: "app:prod", range: "6h", data: ["new"] },
    });
  });

  it("keeps the same held state for fresh data already held", () => {
    expect(resolveHeld(held, { ...view, range: "1h", fresh: held.data }).held).toBe(held);
  });

  it("shows no data for another view", () => {
    const result = resolveHeld(held, { ...view, viewKey: "app:staging", range: "1h" });
    expect(result).toEqual({ data: undefined, held });
    expect(result.held).toBe(held);
    expect(resolveHeld(null, view)).toEqual({ data: undefined, held: null });
  });

  it("uses held data for the same range, even on error", () => {
    const result = resolveHeld(held, { ...view, range: "1h", isError: true });
    expect(result).toEqual({ data: ["old"], held });
    expect(result.held).toBe(held);
  });

  it("uses held data while another range loads", () => {
    const result = resolveHeld(held, view);
    expect(result).toEqual({ data: ["old"], held });
    expect(result.held).toBe(held);
  });

  it("shows no data when another range fails", () => {
    const result = resolveHeld(held, { ...view, isError: true });
    expect(result).toEqual({ data: undefined, held });
    expect(result.held).toBe(held);
  });
});
