import { getErrorMessage } from "@/lib/unkey-client";
import { ConflictErrorResponse } from "@unkey/api/models/errors";

/**
 * `createPortal` and `updatePortal` both report every duplicate as a 409 under
 * the same `Data.Portal.Duplicate` code, so the public detail is the only thing
 * that separates them. There are exactly three, verbatim from:
 *
 * - `v2_portal_create_portal/handler.go:243` and
 *   `v2_portal_update_portal/handler.go:436` — the slug pre-check.
 * - `v2_portal_create_portal/handler.go:274` and
 *   `v2_portal_update_portal/handler.go:476` — the mapping pre-check.
 * - `v2_portal_create_portal/handler.go:163` and
 *   `v2_portal_update_portal/handler.go:297` — the unique-index arm, reached
 *   when a concurrent write wins the key between the pre-check and the write.
 *
 * These are wire-contract strings: one copy, so the two surfaces cannot drift.
 */
export const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";
export const MAPPING_CONFLICT_DETAIL = "That app or keyspace already has a portal.";
export const AMBIGUOUS_CONFLICT_DETAIL = "A portal already exists for that slug, app, or keyspace.";

/**
 * The mapping check spans every workspace, so the portal holding this keyspace
 * may be one this operator cannot see. Telling them to pick another slug would
 * send them round a loop no slug can win.
 */
export const MAPPING_CONFLICT_MESSAGE =
  "This API's keyspace already has a customer portal. It may belong to another workspace. " +
  "Contact support@unkey.com if you think that's wrong.";

/**
 * A 409 can mean the write landed and only the acknowledgement was lost, which
 * a re-read settles. When that re-read itself fails there is nothing to settle
 * it with, and sending the operator off to rename would be wrong if their
 * portal exists.
 */
export const CONFLICT_UNRESOLVED_MESSAGE =
  "We couldn't confirm whether the portal was created. Reload this page to check before " +
  "trying again.";

/**
 * Classifies a portal 409 by its public detail, or returns null when the error
 * is not a portal conflict.
 *
 * `"ambiguous"` is the unique-index arm. It is ambiguous by construction: the
 * server does not name the index it lost, so the caller cannot tell a slug
 * collision from a mapping one and has to resolve it by reading.
 *
 * The response class is checked first: `getErrorMessage` returns a detail for
 * any of eleven error classes, so an unrelated 400 or 500 that happened to
 * carry the same text would otherwise be routed onto the slug field.
 */
export function portalConflict(error: unknown): "slug" | "mapping" | "ambiguous" | null {
  if (!(error instanceof ConflictErrorResponse)) {
    return null;
  }
  const detail = getErrorMessage(error);
  if (detail === SLUG_CONFLICT_DETAIL) {
    return "slug";
  }
  if (detail === MAPPING_CONFLICT_DETAIL) {
    return "mapping";
  }
  if (detail === AMBIGUOUS_CONFLICT_DETAIL) {
    return "ambiguous";
  }
  return null;
}
