import { describe, expect, it } from "vitest";
import { formatUserCode } from "./format-user-code";

describe("formatUserCode", () => {
  it.each(["7k3m9r2x", "7k3m-9r2x"])("groups all eight symbols in %s", (code) => {
    expect(formatUserCode(code)).toBe("7K3M-9R2X");
  });

  it("trims surrounding whitespace", () => {
    expect(formatUserCode("  7K3M-9R2X  ")).toBe("7K3M-9R2X");
  });
});
