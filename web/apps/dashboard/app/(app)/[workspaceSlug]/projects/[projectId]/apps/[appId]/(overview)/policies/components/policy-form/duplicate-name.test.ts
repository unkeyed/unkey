import { describe, expect, it } from "vitest";
import { duplicateNameError } from "./duplicate-name";

const existing = ["keyauth:Checkout", "ratelimit:Public"];

describe("duplicateNameError", () => {
  it("rejects a new policy whose type and name already exist", () => {
    expect(duplicateNameError({ type: "keyauth", name: "Checkout" }, existing, null)).toBe(
      'A Key Auth policy named "Checkout" already exists. Open it to add it to another environment.',
    );
  });

  it("matches on the trimmed name", () => {
    expect(duplicateNameError({ type: "ratelimit", name: " Public " }, existing, null)).toBe(
      'A Rate Limit policy named " Public " already exists. Open it to add it to another environment.',
    );
  });

  it("accepts the same name under another type", () => {
    expect(duplicateNameError({ type: "firewall", name: "Checkout" }, existing, null)).toBe(null);
  });

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
