import { describe, expect, it } from "vitest";
import {
  getUsagePeriodOptions,
  resolveUsagePeriod,
  usagePeriodMonth,
  usagePeriodRange,
} from "./period";

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

describe("usagePeriodRange", () => {
  const now = new Date("2026-10-12T12:00:00Z");

  it("spans the current calendar month and counts days to the reset", () => {
    expect(usagePeriodRange("current", now)).toEqual({
      label: "Oct 1 – 31, 2026",
      daysLeft: 20,
    });
  });

  it("has no days left for the previous month", () => {
    expect(usagePeriodRange("previous", now)).toEqual({
      label: "Sep 1 – 30, 2026",
      daysLeft: null,
    });
  });

  it("counts the last day of the month as one day left", () => {
    expect(usagePeriodRange("current", new Date("2026-10-31T23:00:00Z")).daysLeft).toBe(1);
  });

  it("rolls the previous month back across the year boundary", () => {
    expect(usagePeriodRange("previous", new Date("2027-01-05T12:00:00Z")).label).toBe(
      "Dec 1 – 31, 2026",
    );
  });
});
