import { describe, expect, it } from "vitest";
import { formatOperator } from "./utils";

describe("formatOperator", () => {
  it("labels exact attribute filters as matches", () => {
    expect(formatOperator("is", "attributes")).toBe("matches");
  });
});
