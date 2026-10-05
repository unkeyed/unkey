import { describe, expect, it } from "vitest";
import { getUsagePeriodOptions, resolveUsagePeriod, usagePeriodMonth } from "./period";

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

describe("usagePeriodMonth", () => {
  it("returns the current and previous UTC months", () => {
    const now = new Date("2026-10-05T12:00:00Z");
    expect(usagePeriodMonth("current", now)).toEqual({ year: 2026, month: 10 });
    expect(usagePeriodMonth("previous", now)).toEqual({ year: 2026, month: 9 });
  });

  it("rolls back into the previous year in January", () => {
    expect(usagePeriodMonth("previous", new Date("2026-01-01T00:30:00Z"))).toEqual({
      year: 2025,
      month: 12,
    });
  });
});
