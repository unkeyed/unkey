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
  const view = {
    viewKey: "app:prod",
    range: "6h" as const,
    fresh: undefined,
    isError: false,
    fetchStatus: "fetching" as const,
  };

  it("uses fresh data and holds it", () => {
    const first = resolveHeld(held, { ...view, fresh: ["new"] });
    expect(first).toEqual({
      data: ["new"],
      held: { viewKey: "app:prod", range: "6h", data: ["new"] },
      outdated: false,
    });
    expect(resolveHeld(first.held, { ...view, fresh: first.data }).held).toBe(first.held);
  });

  it("shows no data for another view", () => {
    expect(resolveHeld(held, { ...view, viewKey: "app:staging", range: "1h" })).toEqual({
      data: undefined,
      held,
      outdated: false,
    });
  });

  it("uses held data for the same range, even on error", () => {
    expect(resolveHeld(held, { ...view, range: "1h", isError: true })).toEqual({
      data: ["old"],
      held,
      outdated: false,
    });
  });

  it("marks held data outdated only while another range is in flight", () => {
    expect(resolveHeld(held, view)).toEqual({ data: ["old"], held, outdated: true });
    expect(resolveHeld(held, { ...view, fetchStatus: "paused" }).outdated).toBe(true);
    expect(resolveHeld(held, { ...view, fetchStatus: "idle" }).outdated).toBe(false);
  });

  it("drops the held data when another range fails", () => {
    expect(resolveHeld(held, { ...view, isError: true })).toEqual({
      data: undefined,
      held: null,
      outdated: false,
    });
  });
});
