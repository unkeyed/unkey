import { describe, expect, it } from "vitest";
import { type Grid, type Held, clampedRetryMs, holdKeyOf, resolveHeld } from "./app-metrics";

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

describe("holdKeyOf", () => {
  it("keys held data by app and environment, not by range", () => {
    const scope = { appId: "app_1", environment: { id: "env_1" }, grid: asked };
    const otherRange = { ...scope, grid: { ...asked, range: "6h" as const } };
    expect(holdKeyOf(scope)).toBe("app_1:env_1");
    expect(holdKeyOf(otherRange)).toBe("app_1:env_1");
    expect(holdKeyOf({ appId: "app_1", environment: { kind: "production" } })).toBe(
      "app_1:production",
    );
    expect(holdKeyOf({ appId: "app_2", environment: { id: "env_1" } })).toBe("app_2:env_1");
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
    expect(resolveHeld(held, { ...view, fresh: ["new"], isError: true })).toEqual({
      data: ["new"],
      held: { viewKey: "app:prod", range: "6h", data: ["new"] },
      outdated: false,
    });
  });

  it("keeps the same held state for fresh data already held", () => {
    expect(resolveHeld(held, { ...view, range: "1h", fresh: held.data }).held).toBe(held);
  });

  it("holds fresh data with the same reference again under a new range", () => {
    expect(resolveHeld(held, { ...view, fresh: held.data })).toEqual({
      data: ["old"],
      held: { viewKey: "app:prod", range: "6h", data: ["old"] },
      outdated: false,
    });
  });

  it("shows no data for another view", () => {
    const result = resolveHeld(held, { ...view, viewKey: "app:staging", range: "1h" });
    expect(result).toEqual({ data: undefined, held, outdated: false });
    expect(result.held).toBe(held);
    expect(resolveHeld(null, view)).toEqual({ data: undefined, held: null, outdated: false });
  });

  it("uses held data for the same range, even on error", () => {
    const result = resolveHeld(held, { ...view, range: "1h", isError: true });
    expect(result).toEqual({ data: ["old"], held, outdated: false });
    expect(result.held).toBe(held);
  });

  it("uses held data as outdated while another range loads", () => {
    const result = resolveHeld(held, view);
    expect(result).toEqual({ data: ["old"], held, outdated: true });
    expect(result.held).toBe(held);
  });

  it("uses held data as outdated while the other range fetch is paused", () => {
    expect(resolveHeld(held, { ...view, fetchStatus: "paused" })).toEqual({
      data: ["old"],
      held,
      outdated: true,
    });
  });

  it("uses held data as current while the other range query is idle", () => {
    expect(resolveHeld(held, { ...view, fetchStatus: "idle" })).toEqual({
      data: ["old"],
      held,
      outdated: false,
    });
  });

  it("drops the held data when another range fails", () => {
    const failure = resolveHeld(held, { ...view, isError: true });
    expect(failure).toEqual({ data: undefined, held: null, outdated: false });
    expect(resolveHeld(failure.held, { ...view, isError: true })).toEqual({
      data: undefined,
      held: null,
      outdated: false,
    });
  });
});
