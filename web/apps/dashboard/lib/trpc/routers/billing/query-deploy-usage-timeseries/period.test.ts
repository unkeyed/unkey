import { describe, expect, it } from "vitest";
import { getDeployUsageQueryPeriod } from "./period";

const AUGUST_31 = Date.UTC(2026, 7, 31);
const SEPTEMBER_2 = Date.UTC(2026, 8, 2);
const SEPTEMBER_3 = Date.UTC(2026, 8, 3);
const now = new Date(Date.UTC(2026, 8, 3, 15, 42));

describe("getDeployUsageQueryPeriod", () => {
  it("returns the complete selected month for daily usage", () => {
    expect(getDeployUsageQueryPeriod({ now, monthsAgo: 1 })).toEqual({
      start: Date.UTC(2026, 7, 1),
      end: Date.UTC(2026, 8, 1),
    });
  });

  it("returns one complete historical day for hourly usage", () => {
    expect(getDeployUsageQueryPeriod({ now, monthsAgo: 0, dayStart: SEPTEMBER_2 })).toEqual({
      start: SEPTEMBER_2,
      end: SEPTEMBER_3,
    });
  });

  it("ends the current day at the current time", () => {
    expect(getDeployUsageQueryPeriod({ now, monthsAgo: 0, dayStart: SEPTEMBER_3 })).toEqual({
      start: SEPTEMBER_3,
      end: now.getTime(),
    });
  });

  it("rejects days outside the selected month", () => {
    expect(getDeployUsageQueryPeriod({ now, monthsAgo: 0, dayStart: AUGUST_31 })).toBeNull();
  });
});
