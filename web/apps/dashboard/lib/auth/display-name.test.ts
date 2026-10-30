import { describe, expect, it } from "vitest";
import { displayNameOf } from "./display-name";

describe("displayNameOf", () => {
  it("joins first and last name", () => {
    expect(displayNameOf({ firstName: "Ada", lastName: "Lovelace" })).toBe("Ada Lovelace");
  });

  it("uses whichever name is set", () => {
    expect(displayNameOf({ firstName: null, lastName: "Lovelace" })).toBe("Lovelace");
  });

  it("is null when neither name is set", () => {
    expect(displayNameOf({ firstName: null, lastName: " " })).toBeNull();
  });
});
