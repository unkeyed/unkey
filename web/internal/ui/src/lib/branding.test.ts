import { describe, expect, it } from "vitest";
import { onPrimaryColor } from "./branding";

const LIGHT = "#ffffff";
const DARK = "#0a0a0a";

describe("onPrimaryColor", () => {
  it("returns the dark color for a light brand color", () => {
    expect(onPrimaryColor("#FFFF00")).toBe(DARK);
  });

  it("returns the light color for a dark brand color", () => {
    expect(onPrimaryColor("#18181B")).toBe(LIGHT);
  });

  it("returns the light color for null", () => {
    expect(onPrimaryColor(null)).toBe(LIGHT);
  });

  it("returns the light color for undefined", () => {
    expect(onPrimaryColor(undefined)).toBe(LIGHT);
  });

  it("returns the light color for an unparseable string", () => {
    expect(onPrimaryColor("not-a-color")).toBe(LIGHT);
  });

  it("resolves three-digit shorthand the same as its six-digit form", () => {
    expect(onPrimaryColor("#fff")).toBe(onPrimaryColor("#ffffff"));
    expect(onPrimaryColor("#fff")).toBe(DARK);
  });
});
