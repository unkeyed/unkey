import { describe, expect, it } from "vitest";
import { availableAnalyticsPeriods, defaultAnalyticsPeriodDays } from "./analytics.schema";

const daysOf = (retention: number) => availableAnalyticsPeriods(retention).map((p) => p.days);

describe("availableAnalyticsPeriods", () => {
  it("offers only 24h and 7d for the free-tier 7-day retention", () => {
    expect(daysOf(7)).toEqual([1, 7]);
  });

  it("offers the ladder up to 30d for a 30-day plan", () => {
    expect(daysOf(30)).toEqual([1, 7, 30]);
  });

  it("offers the full ladder for a 90-day plan", () => {
    expect(daysOf(90)).toEqual([1, 7, 30, 90]);
  });

  it("appends an exact-max rung when retention falls between ladder rungs", () => {
    expect(daysOf(45)).toEqual([1, 7, 30, 45]);
    expect(daysOf(60)).toEqual([1, 7, 30, 60]);
  });

  it("appends an exact-max rung above the top ladder rung", () => {
    expect(daysOf(120)).toEqual([1, 7, 30, 90, 120]);
  });

  it("clamps the exact-max rung to the 365-day data ceiling", () => {
    expect(daysOf(1000)).toEqual([1, 7, 30, 90, 365]);
  });

  it("treats non-positive retention as uncapped (full ladder)", () => {
    expect(daysOf(0)).toEqual([1, 7, 30, 90]);
    expect(daysOf(-1)).toEqual([1, 7, 30, 90]);
  });

  it("labels the 1-day window as 24 hours and others as N days", () => {
    const options = availableAnalyticsPeriods(45);
    expect(options[0].label).toBe("Last 24 hours");
    expect(options.at(-1)?.label).toBe("Last 45 days");
  });
});

describe("defaultAnalyticsPeriodDays", () => {
  it("defaults to the 7-day view when retention allows it", () => {
    expect(defaultAnalyticsPeriodDays(7)).toBe(7);
    expect(defaultAnalyticsPeriodDays(30)).toBe(7);
    expect(defaultAnalyticsPeriodDays(90)).toBe(7);
  });

  it("falls back to the widest available window when 7d is out of reach", () => {
    // Retention below 7 (defensive — plans don't go this low) has no 7d rung.
    expect(defaultAnalyticsPeriodDays(3)).toBe(3);
  });
});
