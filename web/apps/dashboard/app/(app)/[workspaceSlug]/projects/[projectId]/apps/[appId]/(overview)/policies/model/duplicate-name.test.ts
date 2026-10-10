import { describe, expect, it } from "vitest";
import { duplicateNameError } from "./duplicate-name";

const existing = ["keyauth:Checkout", "ratelimit:Public"];

describe("duplicateNameError", () => {
  it("accepts an edited policy that keeps its own name", () => {
    expect(
      duplicateNameError({ type: "keyauth", name: "Checkout" }, existing, {
        type: "keyauth",
        name: "Checkout",
      }),
    ).toBe(null);
  });

  it("rejects an edited policy renamed onto another policy", () => {
    expect(
      duplicateNameError({ type: "ratelimit", name: "Public" }, existing, {
        type: "ratelimit",
        name: "Internal",
      }),
    ).toBe(
      'A Rate Limit policy named "Public" already exists. Open it to add it to another environment.',
    );
  });
});
