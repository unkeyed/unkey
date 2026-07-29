import { describe, expect, it } from "vitest";
import { formatCompactQuantity } from "./fmt";

describe("formatCompactQuantity", () => {
  it("compacts thousands with a lowercase k to ~3 significant figures", () => {
    expect(formatCompactQuantity(10_386.7)).toBe("10.4k");
    expect(formatCompactQuantity(2_596.7)).toBe("2.6k");
    expect(formatCompactQuantity(12_345)).toBe("12.3k");
  });

  it("leaves sub-thousand values as-is", () => {
    expect(formatCompactQuantity(259.7)).toBe("259.7");
    expect(formatCompactQuantity(43.3)).toBe("43.3");
    expect(formatCompactQuantity(0)).toBe("0");
  });

  it("keeps uppercase M for millions", () => {
    expect(formatCompactQuantity(1_250_000)).toBe("1.3M");
  });
});
