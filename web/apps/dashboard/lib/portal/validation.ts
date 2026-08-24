import { z } from "zod";

/**
 * Client-side mirrors of the server's portal validators. They exist because the
 * generated OpenAPI schema is not a faithful copy of the Go rules: the slug
 * `pattern` cannot express the consecutive-hyphen rejection that
 * `pkg/validation/slug.go` applies on top of it. Messages match the server
 * verbatim except for the slug, where the form uses friendlier wording; the
 * server's wording is still what the API returns if a request skips the form.
 *
 * Sources: `pkg/validation/slug.go`, `svc/api/internal/portal/portal.go`.
 */

export const SLUG_MIN_LENGTH = 3;
export const SLUG_MAX_LENGTH = 64;
const DISPLAY_NAME_MAX_LENGTH = 64;
const LOGO_URL_MAX_LENGTH = 500;

/**
 * Covers the same rules as `validation.ErrMsgInvalidSlug` but reworded to fit on
 * one line under the input.
 */
export const INVALID_SLUG_MESSAGE =
  "Use 3-64 lowercase letters, numbers, and hyphens (not at the start, end, or doubled).";

// The server enforces this one through OpenAPI validation rather than a named
// message, so the wording here is ours.
export const INVALID_DISPLAY_NAME_MESSAGE =
  "Enter a name of up to 64 characters, with no leading or trailing spaces.";

/** Verbatim from `portal.ErrMsgInvalidLogoURL`. */
export const INVALID_LOGO_URL_MESSAGE =
  "logoUrl must be an absolute https:// URL of at most 500 characters.";

/** Verbatim from `portal.ErrMsgInvalidColor`. */
export const INVALID_COLOR_MESSAGE =
  "primaryColor must be a six-digit hex colour, for example #6366f1.";

const slugPattern = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/;
const hexColorPattern = /^#[0-9a-fA-F]{6}$/;

export const portalSlugSchema = z
  .string()
  .min(SLUG_MIN_LENGTH, INVALID_SLUG_MESSAGE)
  .max(SLUG_MAX_LENGTH, INVALID_SLUG_MESSAGE)
  .regex(slugPattern, INVALID_SLUG_MESSAGE)
  // The regex above matches `a--b`; the server rejects it separately, so this
  // rule has to be separate here too.
  .refine((slug) => !slug.includes("--"), INVALID_SLUG_MESSAGE);

// Mirrors the spec's `^\S(.*\S)?$`: at least one character, and no leading or
// trailing whitespace.
const displayNamePattern = /^\S(.*\S)?$/;

export const portalDisplayNameSchema = z
  .string()
  .max(DISPLAY_NAME_MAX_LENGTH, INVALID_DISPLAY_NAME_MESSAGE)
  .regex(displayNamePattern, INVALID_DISPLAY_NAME_MESSAGE);

// `new URL("https:///logo.png")` normalizes the empty authority away and reports
// host "logo.png", where Go's url.Parse leaves the host empty and the server
// rejects it. Requiring a non-slash character right after the scheme keeps the
// two parsers agreeing, so the form never accepts a URL the API will refuse.
const absoluteHttpsPattern = /^https:\/\/[^/]/;

function isValidLogoUrl(raw: string): boolean {
  if (raw.length > LOGO_URL_MAX_LENGTH || !absoluteHttpsPattern.test(raw)) {
    return false;
  }
  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    return false;
  }
  // No file-extension requirement: the server accepts extensionless CDN URLs.
  return parsed.protocol === "https:" && parsed.host !== "";
}

/**
 * An empty value passes: clearing branding is legal, and the caller maps empty
 * to the API's `null` rather than sending `""`, which the server rejects.
 */
export const logoUrlSchema = z
  .string()
  .refine((raw) => raw === "" || isValidLogoUrl(raw), INVALID_LOGO_URL_MESSAGE);

/** Six digits only. Three-digit shorthand and named colours are rejected. */
export function isHexColor(value: string): boolean {
  return hexColorPattern.test(value);
}

export const primaryColorSchema = z
  .string()
  .refine((raw) => raw === "" || isHexColor(raw), INVALID_COLOR_MESSAGE);
