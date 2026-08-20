import { getErrorMessage } from "@/lib/unkey-client";
import { ConflictErrorResponse } from "@unkey/api/models/errors";

/**
 * `createPortal` and `updatePortal` both report these as 409s under the same
 * `Data.Portal.Duplicate` code, so the public detail is the only thing that
 * separates them. Verbatim from `svc/api/routes/v2_portal_create_portal/handler.go`
 * and `svc/api/routes/v2_portal_update_portal/handler.go`.
 *
 * These are wire-contract strings: one copy, so the two surfaces cannot drift.
 */
export const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";
export const MAPPING_CONFLICT_DETAIL = "That app or keyspace already has a portal.";

/**
 * The mapping check spans every workspace, so the portal holding this keyspace
 * may be one this operator cannot see. Telling them to pick another slug would
 * send them round a loop no slug can win.
 */
export const MAPPING_CONFLICT_MESSAGE =
  "This API's keyspace already has a customer portal. It may belong to another workspace. " +
  "Contact support@unkey.com if you think that's wrong.";

/**
 * Classifies a portal 409 by its public detail, or returns null when the error
 * is not a portal conflict.
 *
 * The response class is checked first: `getErrorMessage` returns a detail for
 * any of eleven error classes, so an unrelated 400 or 500 that happened to
 * carry the same text would otherwise be routed onto the slug field.
 */
export function portalConflict(error: unknown): "slug" | "mapping" | null {
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
  return null;
}
