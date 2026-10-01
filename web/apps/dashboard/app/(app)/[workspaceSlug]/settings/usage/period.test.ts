import { describe, expect, it } from "vitest";
import { getUsagePeriods, resolveUsagePeriod } from "./period";

describe("getUsagePeriods", () => {
  it("returns the current and previous UTC months using MMYYYY values", () => {
    const periods = getUsagePeriods(new Date("2026-01-15T12:00:00Z"));

    expect(periods).toEqual([
      {
        value: "012026",
        label: "Current month",
        monthsAgo: 0,
        start: Date.UTC(2026, 0, 1),
        end: Date.parse("2026-01-15T12:00:00Z"),
      },
      {
        value: "122025",
        label: "Last month",
        monthsAgo: 1,
        start: Date.UTC(2025, 11, 1),
        end: Date.UTC(2026, 0, 1),
      },
    ]);
  });
});

describe("resolveUsagePeriod", () => {
  it("falls back to the current month for an unsupported query value", () => {
    const periods = getUsagePeriods(new Date("2026-01-15T12:00:00Z"));

    expect(resolveUsagePeriod("112025", periods)).toBe(periods[0]);
  });
});
