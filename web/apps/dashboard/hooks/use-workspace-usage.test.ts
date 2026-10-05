import { describe, expect, it } from "vitest";
import { previousUsageMonth } from "./use-workspace-usage";

describe("previousUsageMonth", () => {
  it("returns the previous UTC month as YYYY-MM", () => {
    expect(previousUsageMonth(new Date("2026-10-05T12:00:00Z"))).toBe("2026-09");
  });

  it("rolls back into the previous year in January", () => {
    expect(previousUsageMonth(new Date("2026-01-01T00:30:00Z"))).toBe("2025-12");
  });
});
