import { ConflictErrorResponse } from "@unkey/api/models/errors";
import { getErrorMessage } from "@/lib/unkey-client";

// Every portal duplicate comes back as a 409 under the same
// `Data.Portal.Duplicate` code, so the public detail is all that separates
// them. Copied verbatim from the create and update handlers in
// `svc/api/routes/v2_portal_{create,update}_portal`; changing one side breaks
// the match.
export const SLUG_CONFLICT_DETAIL = "That slug is already in use. Choose a different slug.";
export const MAPPING_CONFLICT_DETAIL = "That app or keyspace already has a portal.";
export const AMBIGUOUS_CONFLICT_DETAIL = "A portal already exists for that slug, app, or keyspace.";

// The mapping check spans every workspace, so the portal holding this keyspace
// may be invisible to this operator and no slug they pick can win.
export const MAPPING_CONFLICT_MESSAGE =
  "This API's keyspace already has a customer portal. It may belong to another workspace. " +
  "Contact support@unkey.com if you think that's wrong.";

// A 409 can mean the write landed and only the acknowledgement was lost. A
// re-read settles that; when the re-read also fails, nothing does.
export const CONFLICT_UNRESOLVED_MESSAGE =
  "We couldn't confirm whether the portal was created. Reload this page to check before " +
  "trying again.";

/**
 * Classifies a portal 409 by its public detail, or returns null when the error
 * is not a portal conflict.
 *
 * `"ambiguous"` is the unique-index arm: the server does not name the index it
 * lost, so the caller has to resolve it by reading.
 *
 * The response class is checked first because `getErrorMessage` returns a
 * detail for every error class, so an unrelated status carrying the same text
 * would otherwise be routed onto the slug field.
 */
// Classifies a 409 into which uniqueness constraint it hit, or null when the
// error is not a portal duplicate.
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
