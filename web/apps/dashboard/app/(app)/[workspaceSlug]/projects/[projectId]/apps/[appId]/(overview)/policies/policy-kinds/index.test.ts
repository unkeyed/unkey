import { describe, expect, it } from "vitest";
import { hasConfigErrors } from "./index";

describe("hasConfigErrors", () => {
  it("is false when only the name or the conditions are invalid", () => {
    expect(
      hasConfigErrors({
        name: { type: "too_small", message: "Name is required" },
        matchConditions: { type: "custom", message: "Add at least one condition" },
      }),
    ).toBe(false);
  });

  it("is true when a type config field is invalid", () => {
    expect(
      hasConfigErrors({
        keyspaceIds: { type: "too_small", message: "Select at least one keyspace" },
      }),
    ).toBe(true);
  });
});
