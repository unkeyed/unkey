import { describe, expect, it } from "vitest";
import {
  INVALID_COLOR_MESSAGE,
  INVALID_LOGO_URL_MESSAGE,
  INVALID_SLUG_MESSAGE,
  logoUrlSchema,
  portalSlugSchema,
  primaryColorSchema,
} from "./validation";

const repeat = (count: number): string => "a".repeat(count);

describe("portalSlugSchema", () => {
  it.each(["abc", "my-portal", "a1b", repeat(64)])("accepts %s", (slug) => {
    expect(portalSlugSchema.safeParse(slug).success).toBe(true);
  });

  it.each(["ab", repeat(65), "My-Portal", "my_portal", "-abc", "abc-", "a--b"])(
    "rejects %s",
    (slug) => {
      const result = portalSlugSchema.safeParse(slug);
      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.issues[0].message).toBe(INVALID_SLUG_MESSAGE);
      }
    },
  );
});

describe("logoUrlSchema", () => {
  it("accepts an https url with no file extension", () => {
    expect(logoUrlSchema.safeParse("https://cdn.example.com/logo").success).toBe(true);
  });

  it("accepts an empty value, because clearing branding is legal", () => {
    expect(logoUrlSchema.safeParse("").success).toBe(true);
  });

  it.each([
    ["a non-https scheme", "http://example.com/logo.png"],
    [
      "a url over 500 characters",
      `https://example.com/${repeat(501 - "https://example.com/".length)}`,
    ],
    ["an unparseable string", "not a url at all"],
    ["a scheme with no host", "https://"],
    // Go's url.Parse leaves the host empty here and the server rejects it, but
    // new URL() normalizes it to host "logo.png". Without the guard the form
    // would accept a value the API refuses.
    ["a collapsed empty authority", "https:///logo.png"],
    // The logo URL is rendered as an <img src> in the portal, so a scheme that
    // can carry executable or inline content must stay rejected. Pinned here so
    // a future relaxation of the https-only rule cannot pass silently.
    ["a javascript: url", "javascript:alert(1)"],
    [
      "a data: url",
      "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4=",
    ],
  ])("rejects %s", (_label, raw) => {
    const result = logoUrlSchema.safeParse(raw);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].message).toBe(INVALID_LOGO_URL_MESSAGE);
    }
  });
});

describe("primaryColorSchema", () => {
  it.each(["#18181B", "#18181b"])("accepts %s", (color) => {
    expect(primaryColorSchema.safeParse(color).success).toBe(true);
  });

  it("accepts an empty value, because clearing branding is legal", () => {
    expect(primaryColorSchema.safeParse("").success).toBe(true);
  });

  it.each(["#181", "red", "18181B"])("rejects %s", (color) => {
    const result = primaryColorSchema.safeParse(color);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].message).toBe(INVALID_COLOR_MESSAGE);
    }
  });
});
