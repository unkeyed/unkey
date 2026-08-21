/**
 * Derives a portal slug candidate from an API name.
 *
 * The output is only a prefill: the operator can edit it before submitting. But
 * it must never be a value `portalSlugSchema` rejects, or the dialog would open
 * already in an error state through no fault of the operator. Names that carry
 * fewer than three legal characters — `"AB"`, `"!!!"`, a name written entirely
 * in a non-Latin script — fall back to a derived candidate rather than leaving
 * the field empty.
 */

import { SLUG_MAX_LENGTH, SLUG_MIN_LENGTH } from "./validation";

/** Used when the name contributes nothing a slug can be built from. */
const FALLBACK_SLUG = "portal";

const COMBINING_MARKS = /\p{M}/gu;
const ILLEGAL_RUN = /[^a-z0-9]+/g;
const EDGE_HYPHENS = /^-+|-+$/g;

export function slugifyPortalName(name: string): string {
  const base = name
    // NFD splits an accented letter into its base letter plus a combining mark,
    // so dropping the marks transliterates "Café" to "cafe" instead of losing
    // the whole character.
    .normalize("NFD")
    .replace(COMBINING_MARKS, "")
    .toLowerCase()
    // One hyphen per run of illegal characters, which is also what keeps
    // whitespace runs from becoming consecutive hyphens.
    .replace(ILLEGAL_RUN, "-")
    .replace(EDGE_HYPHENS, "");

  // Trim first, then re-strip: slicing can land mid-separator and leave a
  // trailing hyphen, which the server rejects.
  const truncated = base.slice(0, SLUG_MAX_LENGTH).replace(EDGE_HYPHENS, "");

  if (truncated.length >= SLUG_MIN_LENGTH) {
    return truncated;
  }
  return truncated === "" ? FALLBACK_SLUG : `${truncated}-${FALLBACK_SLUG}`;
}
