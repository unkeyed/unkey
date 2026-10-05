import { describe, expect, it } from "vitest";
import { previousUsagePeriod } from "./use-workspace-usage";

describe("previousUsagePeriod", () => {
  it("returns the previous UTC month", () => {
    expect(previousUsagePeriod(new Date("2026-10-05T12:00:00Z"))).toEqual({ year: 2026, month: 9 });
  });

  it("rolls back into the previous year in January", () => {
    expect(previousUsagePeriod(new Date("2026-01-01T00:30:00Z"))).toEqual({
      year: 2025,
      month: 12,
    });
  });
});
