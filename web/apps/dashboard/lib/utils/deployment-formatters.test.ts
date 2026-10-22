import { describe, expect, it } from "vitest";
import { formatTooltipPercent } from "./deployment-formatters";

describe("formatTooltipPercent", () => {
  it("floors a non-zero share below 0.01% instead of printing zero", () => {
    expect(formatTooltipPercent(0.002)).toBe("<0.01%");
    expect(formatTooltipPercent(0)).toBe("0%");
  });

  it("keeps its precision steps from 0.01% up", () => {
    expect(formatTooltipPercent(0.01)).toBe("0.01%");
    expect(formatTooltipPercent(2)).toBe("2.0%");
    expect(formatTooltipPercent(42.4)).toBe("42%");
  });
});
