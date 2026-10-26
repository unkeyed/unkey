import { describe, expect, it } from "vitest";
import { type Grid, clampedRetryMs } from "./app-metrics";

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
