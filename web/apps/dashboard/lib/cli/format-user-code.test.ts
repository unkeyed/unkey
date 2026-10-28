import { describe, expect, it } from "vitest";
import { formatUserCode } from "./format-user-code";

describe("formatUserCode", () => {
  it("spaces a device code", () => {
    expect(formatUserCode("zhvg-sxqm")).toBe("Z H V G - S X Q M");
  });

  it("trims surrounding whitespace", () => {
    expect(formatUserCode("  ABCD-EFGH  ")).toBe("A B C D - E F G H");
  });
});
