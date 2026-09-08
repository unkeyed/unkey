/**
 * Derives a portal slug candidate from an API name.
 *
 * The output is a prefill the operator can edit, but it must never be a value
 * `portalSlugSchema` rejects, or the dialog opens already in an error state.
 */

import { SLUG_MAX_LENGTH, SLUG_MIN_LENGTH } from "./validation";

const FALLBACK_SLUG = "portal";

const COMBINING_MARKS = /\p{M}/gu;
const ILLEGAL_RUN = /[^a-z0-9]+/g;
const EDGE_HYPHENS = /^-+|-+$/g;

// Derives a slug candidate from a name. Only a prefill: the operator can edit it,
// but it must never be a value `portalSlugSchema` rejects.
export function slugifyPortalName(name: string): string {
  const base = name
    // NFD splits an accented letter into base plus combining mark, so dropping
    // the marks gives "cafe" rather than losing the character entirely.
    .normalize("NFD")
    .replace(COMBINING_MARKS, "")
    .toLowerCase()
    .replace(ILLEGAL_RUN, "-")
    .replace(EDGE_HYPHENS, "");

  // Re-strip after slicing: a cut can land mid-separator and leave a trailing
  // hyphen, which the server rejects.
  const truncated = base.slice(0, SLUG_MAX_LENGTH).replace(EDGE_HYPHENS, "");

  if (truncated.length >= SLUG_MIN_LENGTH) {
    return truncated;
  }
  return truncated === "" ? FALLBACK_SLUG : `${truncated}-${FALLBACK_SLUG}`;
}
