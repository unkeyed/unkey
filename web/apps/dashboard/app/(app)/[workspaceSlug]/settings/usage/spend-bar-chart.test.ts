import { describe, expect, it } from "vitest";
import { formatAxisCents, spendTicks } from "./spend-bar-chart";

describe("spendTicks", () => {
  it("keeps a frame when nothing was spent", () => {
    expect(spendTicks(0)).toEqual([0, 100]);
  });

  it("steps in cents under four dollars", () => {
    expect(spendTicks(350)).toEqual([0, 88, 176, 264, 352]);
  });

  it("steps in whole dollars sized to the peak", () => {
    expect(spendTicks(11_105)).toEqual([0, 2800, 5600, 8400, 11_200]);
  });
});

describe("formatAxisCents", () => {
  it("prints whole dollars without cents and everything else to the cent", () => {
    expect(formatAxisCents(0)).toBe("$0");
    expect(formatAxisCents(42)).toBe("$0.42");
    expect(formatAxisCents(2800)).toBe("$28");
    expect(formatAxisCents(123_400)).toBe("$1,234");
  });

  it("labels cent steps with their exact value", () => {
    expect(spendTicks(350).map(formatAxisCents)).toEqual([
      "$0",
      "$0.88",
      "$1.76",
      "$2.64",
      "$3.52",
    ]);
  });
});
