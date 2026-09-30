import { describe, expect, it } from "vitest";
import { slugifyPortalName } from "./slugify";
import { portalSlugSchema } from "./validation";

describe("slugifyPortalName", () => {
  it("lowercases and hyphenates a plain name", () => {
    expect(slugifyPortalName("Payments API")).toBe("payments-api");
  });

  it("collapses whitespace runs to a single hyphen", () => {
    expect(slugifyPortalName("My  Spaced   Name")).toBe("my-spaced-name");
  });

  it("transliterates non-ASCII rather than emitting an invalid slug", () => {
    expect(slugifyPortalName("Café API")).toBe("cafe-api");
  });

  it("falls back when the name yields fewer than three legal characters", () => {
    // Two legal characters and none at all both have to produce something the
    // dialog can prefill; blocking on an empty field is not an option.
    expect(portalSlugSchema.safeParse(slugifyPortalName("AB")).success).toBe(true);
    expect(portalSlugSchema.safeParse(slugifyPortalName("!!!")).success).toBe(true);
    expect(portalSlugSchema.safeParse(slugifyPortalName("")).success).toBe(true);
  });

  it("truncates an over-long name without leaving a trailing hyphen", () => {
    // The 65th character is the separator, so a naive slice would end in "-".
    const name = `${"a".repeat(63)} ${"b".repeat(20)}`;
    expect(slugifyPortalName(name)).toBe("a".repeat(63));
  });

  it("never emits consecutive hyphens", () => {
    expect(slugifyPortalName("Acme -- Portal")).toBe("acme-portal");
  });

  it("produces a valid slug for every case above", () => {
    const names = [
      "Payments API",
      "My  Spaced   Name",
      "Café API",
      "AB",
      "!!!",
      "",
      `${"a".repeat(63)} ${"b".repeat(20)}`,
      "Acme -- Portal",
      "  -leading and trailing-  ",
      "日本語",
      "API",
    ];
    for (const name of names) {
      const result = portalSlugSchema.safeParse(slugifyPortalName(name));
      expect(result.success, `${name} -> ${slugifyPortalName(name)}`).toBe(true);
    }
  });
});
