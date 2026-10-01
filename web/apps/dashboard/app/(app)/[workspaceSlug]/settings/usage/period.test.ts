import { describe, expect, it } from "vitest";
import { getUsagePeriods, resolveUsagePeriod } from "./period";

describe("getUsagePeriods", () => {
  it("returns the previous and current UTC months", () => {
    const periods = getUsagePeriods(new Date("2026-01-15T12:00:00Z"));

    expect(periods).toEqual([
      {
        value: "previous",
        label: "December 2025",
        monthsAgo: 1,
        start: Date.UTC(2025, 11, 1),
        end: Date.UTC(2026, 0, 1),
      },
      {
        value: "current",
        label: "January 2026",
        monthsAgo: 0,
        start: Date.UTC(2026, 0, 1),
        end: Date.parse("2026-01-15T12:00:00Z"),
      },
    ]);
  });
});

describe("resolveUsagePeriod", () => {
  it("falls back to the current month for an unsupported query value", () => {
    const periods = getUsagePeriods(new Date("2026-01-15T12:00:00Z"));

    expect(resolveUsagePeriod("unsupported", periods)).toBe(periods[1]);
  });
});
