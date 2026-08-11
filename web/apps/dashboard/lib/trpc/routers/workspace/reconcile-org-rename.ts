import { insertAuditLogs } from "@/lib/audit";
import { auth as authClient } from "@/lib/auth/server";
import type { Organization } from "@/lib/auth/types";
import { and, db, eq, schema, sql } from "@/lib/db";
import { logOperation } from "@/lib/logging/structured-logger";
import { errorLogDetail } from "../utils/errors";

/**
 * The state the caller must report on after a failed auth-provider org rename:
 * - "rename-in-effect": the org holds the requested name (the provider applied
 *   it and lost the response); report success.
 * - "reverted": the rename left no effect; a plain failure reply is truthful.
 * - "superseded": a concurrent rename already overwrote this one, and nothing
 *   was reverted; report a conflict, not "nothing changed".
 * - "left-at-requested-name": the DB holds the requested name while the org
 *   may not (org unreadable, revert failed, or same-name repair); a retry of
 *   the mutation converges both sides.
 */
export type ReconcileOutcome =
  | "rename-in-effect"
  | "reverted"
  | "superseded"
  | "left-at-requested-name";

/**
 * Byte-wise equality predicate for workspaces.name. The column's default
 * collation is case/accent insensitive and utf8mb4_bin is PAD SPACE; either
 * would let a concurrent rename to a near-variant slip past an optimistic
 * guard. utf8mb4_0900_bin (NO PAD, byte-ordered) matches the JS strict
 * equality used everywhere else in the rename flow.
 */
export function workspaceNameByteEquals(name: string) {
  return sql`${schema.workspaces.name} COLLATE utf8mb4_0900_bin = ${name}`;
}

/**
 * True when a name read back from the auth provider matches ours, allowing
 * for Unicode normalization form and edge whitespace. A rename the provider
 * applied in canonicalized form must not be mistaken for one that never
 * applied: reverting over a difference the provider will always reintroduce
 * would leave the two sides permanently divergent.
 */
export function providerNameMatches(providerName: string, name: string): boolean {
  return providerName.normalize("NFC").trim() === name.normalize("NFC").trim();
}

/**
 * Repairs the split state left when the DB workspace rename committed but the
 * auth-provider org rename rejected. Reads the org to decide a safe direction,
 * then compensates the DB side only, since that is the side we control.
 */
export async function reconcileFailedOrgRename(params: {
  orgId: string;
  workspaceId: string;
  requestedName: string;
  previousName: string;
  actorId: string;
  audit: { location: string; userAgent?: string };
}): Promise<ReconcileOutcome> {
  const { orgId, workspaceId, requestedName, previousName, actorId, audit } = params;

  let org: Organization;
  try {
    org = await authClient.getOrg(orgId);
  } catch (readErr) {
    // With the org unreadable there is no safe direction to repair in.
    logOperation(
      "error",
      "Unable to read the auth-provider org after a failed rename; the org name may be out of sync with the workspace name",
      {
        workspace_id: workspaceId,
        error_detail: errorLogDetail(readErr),
      },
    );
    return "left-at-requested-name";
  }

  if (providerNameMatches(org.name, requestedName)) {
    return "rename-in-effect";
  }

  if (previousName === requestedName) {
    // Same-name repair: nothing to revert, and "reverting" X to X would
    // write a nonsensical audit entry.
    return "left-at-requested-name";
  }

  let reverted: boolean;
  try {
    reverted = await db.transaction(async (tx) => {
      // The byte-wise guard keeps the revert from clobbering a concurrent
      // rename that already moved the workspace past requestedName.
      const revert = await tx
        .update(schema.workspaces)
        .set({
          name: previousName,
        })
        .where(and(eq(schema.workspaces.id, workspaceId), workspaceNameByteEquals(requestedName)));

      if (revert[0].affectedRows === 0) {
        return false;
      }

      await insertAuditLogs(tx, {
        workspaceId,
        actor: { type: "user", id: actorId },
        event: "workspace.update",
        description: `Reverted name from ${requestedName} back to ${previousName} after the organization rename failed`,
        resources: [
          {
            type: "workspace",
            id: workspaceId,
            name: previousName,
          },
        ],
        context: {
          location: audit.location,
          userAgent: audit.userAgent,
        },
      });

      return true;
    });
  } catch (revertErr) {
    logOperation(
      "error",
      "Failed to revert the workspace name after the organization rename failed; it may be out of sync with the org name",
      {
        workspace_id: workspaceId,
        error_detail: errorLogDetail(revertErr),
      },
    );
    return "left-at-requested-name";
  }

  return reverted ? "reverted" : "superseded";
}
