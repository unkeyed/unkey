import { describe, expect, it } from "vitest";
import { getUsagePeriodOptions, resolveUsagePeriod } from "./period";

describe("getUsagePeriodOptions", () => {
  it("returns the previous and current UTC months", () => {
    const periods = getUsagePeriodOptions(new Date("2026-01-15T12:00:00Z"));

    expect(periods).toEqual([
      {
        value: "previous",
        label: "December 2025",
      },
      {
        value: "current",
        label: "January 2026",
      },
    ]);
  });
});

describe("resolveUsagePeriod", () => {
  it("falls back to the current month for an unsupported query value", () => {
    expect(resolveUsagePeriod("unsupported")).toBe("current");
  });
});
