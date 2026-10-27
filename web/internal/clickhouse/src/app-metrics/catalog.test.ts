import { describe, expect, it } from "vitest";
import { rollupsServe } from "./catalog";
import { latestGrid, planGrid } from "./grid";

const now = Date.parse("2026-10-19T12:45:20Z");
const day = 24 * 60 * 60 * 1000;

describe("rollupsServe", () => {
  it("serves every range that ends now", () => {
    expect(rollupsServe(latestGrid("30d", now), now)).toBe(true);
    expect(rollupsServe(latestGrid("1h", now), now)).toBe(true);
  });

  it("refuses a 15m grid older than the minute rollup keeps", () => {
    expect(rollupsServe(planGrid("24h", now - 40 * day), now)).toBe(false);
  });

  it("refuses a grid older than the hourly rollup keeps", () => {
    expect(rollupsServe(planGrid("30d", now - 70 * day), now)).toBe(false);
  });
});
