import { insertAuditLogs } from "@/lib/audit";
import { auth as authClient } from "@/lib/auth/server";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { logOperation } from "@/lib/logging/structured-logger";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, requireWorkspaceAdmin, withRatelimit, workspaceProcedure } from "../../trpc";
import { errorLogDetail, wrapUnexpectedError } from "../utils/errors";
import {
  providerNameMatches,
  reconcileFailedOrgRename,
  workspaceNameByteEquals,
} from "./reconcile-org-rename";

const genericErrorMessage =
  "We are unable to update the workspace name. Please try again or contact support@unkey.com";

const conflictErrorMessage =
  "The workspace name changed while your request was in flight. Refresh to see the current name and try again.";

export const changeWorkspaceName = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      // Trimmed before the length checks so whitespace cannot pad a short
      // name or disguise a same-name repair submit.
      name: z
        .string()
        .trim()
        .min(3, "Workspace names must contain at least 3 characters")
        .max(50, "Workspace names must contain less than 50 characters"),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }): Promise<{ updated: boolean }> => {
    if (input.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Invalid workspace ID",
      });
    }

    const previousName = ctx.workspace.name;

    // A same-name submit is a repair attempt: nothing to write on the DB
    // side, but the auth-provider org may still be out of sync from an
    // earlier partial failure.
    const isRepairAttempt = input.name === previousName;

    // The repair audit's "after an earlier partial rename" wording is only
    // truthful when the mismatch was actually observed; a failed read proves
    // nothing.
    let orgConfirmedOutOfSync = false;

    if (isRepairAttempt) {
      // No transaction guards the repair push, so re-read the live row: a
      // stale context must not repair a workspace that was concurrently
      // renamed or soft-deleted. Runs before the no-op check so "already up
      // to date" is never claimed from a stale view.
      const workspace = await db.query.workspaces.findFirst({
        columns: { name: true },
        where: and(
          eq(schema.workspaces.id, input.workspaceId),
          isNull(schema.workspaces.deletedAtM),
        ),
      });
      if (workspace === undefined) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found",
        });
      }
      if (workspace.name !== input.name) {
        throw new TRPCError({
          code: "CONFLICT",
          message: conflictErrorMessage,
        });
      }

      // When the org already agrees there is nothing to repair: an idle Save
      // click becomes a true no-op. An unreadable org falls through to the
      // push, which is idempotent.
      try {
        const org = await authClient.getOrg(ctx.tenant.id);
        if (providerNameMatches(org.name, input.name)) {
          return { updated: false };
        }
        orgConfirmedOutOfSync = true;
      } catch (readErr) {
        logOperation("warn", "Unable to read the auth-provider org before a repair push", {
          workspace_id: ctx.workspace.id,
          error_detail: errorLogDetail(readErr),
        });
      }
    } else {
      // No external call inside the transaction: the row lock would span the
      // provider's latency, and Vitess kills transactions that outlive its
      // timeout.
      await db
        .transaction(async (tx) => {
          // Byte-wise guard: a concurrent rename must surface as CONFLICT,
          // and previousName must be the row's real prior value in case the
          // reconcile below has to revert to it.
          const renamed = await tx
            .update(schema.workspaces)
            .set({
              name: input.name,
            })
            .where(
              and(
                eq(schema.workspaces.id, input.workspaceId),
                isNull(schema.workspaces.deletedAtM),
                workspaceNameByteEquals(previousName),
              ),
            );

          if (renamed[0].affectedRows === 0) {
            // Zero rows means either a concurrent rename or a concurrent
            // soft delete; a re-read tells them apart so a deleted workspace
            // is not misreported as a name conflict.
            const live = await tx.query.workspaces.findFirst({
              columns: { id: true },
              where: and(
                eq(schema.workspaces.id, input.workspaceId),
                isNull(schema.workspaces.deletedAtM),
              ),
            });
            if (live === undefined) {
              throw new TRPCError({
                code: "NOT_FOUND",
                message: "Workspace not found",
              });
            }
            throw new TRPCError({
              code: "CONFLICT",
              message: conflictErrorMessage,
            });
          }

          await insertAuditLogs(tx, {
            workspaceId: ctx.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "workspace.update",
            description: `Changed name from ${previousName} to ${input.name}`,
            resources: [
              {
                type: "workspace",
                id: ctx.workspace.id,
                name: input.name,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        })
        .catch((err) => {
          wrapUnexpectedError(err, genericErrorMessage);
        });
    }

    // A repair changes only the provider, so this audit entry is its sole
    // record. Best-effort: the org rename has already applied, and a failed
    // audit write must not fail the mutation it records.
    const auditRepairSync = async () => {
      try {
        await insertAuditLogs(db, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: orgConfirmedOutOfSync
            ? `Synchronized the organization name to ${input.name} after an earlier partial rename`
            : `Re-applied the organization name ${input.name} at the auth provider; its previous state could not be read`,
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: input.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      } catch (auditErr) {
        logOperation("error", "Failed to write the audit entry for an organization-name repair", {
          workspace_id: ctx.workspace.id,
          error_detail: errorLogDetail(auditErr),
        });
      }
    };

    // The org rename runs only after the DB rename is durable, so a failure
    // leaves the DB in a known state. Two accepted, self-repairing races
    // remain: concurrent renames can reach the provider out of commit order,
    // and a crash between commit and push leaves an unlogged divergence
    // (closing that needs a durable outbox); the same-name repair path
    // re-converges both on the next submit.
    try {
      await authClient.updateOrg({
        id: ctx.tenant.id,
        name: input.name,
      });
    } catch (err) {
      // The expected-error path logs only the client-facing message, so the
      // provider's failure reason is recorded here.
      logOperation("warn", "Auth-provider org rename failed; reconciling the workspace name", {
        workspace_id: ctx.workspace.id,
        error_detail: errorLogDetail(err),
      });
      const outcome = await reconcileFailedOrgRename({
        orgId: ctx.tenant.id,
        workspaceId: ctx.workspace.id,
        requestedName: input.name,
        previousName,
        actorId: ctx.user.id,
        audit: ctx.audit,
      });
      switch (outcome) {
        case "rename-in-effect": {
          // The provider holds the name despite the errored push, so the
          // sync applied and this mutation is a success.
          if (isRepairAttempt) {
            await auditRepairSync();
          }
          return { updated: true };
        }
        case "reverted": {
          // Always throws; the `throw` keyword satisfies fallthrough lint.
          throw wrapUnexpectedError(err, genericErrorMessage);
        }
        case "superseded": {
          // A concurrent rename overwrote this one before the revert ran, so
          // the race surfaces as a conflict, not as "nothing changed".
          throw new TRPCError({
            code: "CONFLICT",
            message:
              "The workspace was renamed again while this request was in flight. Refresh to see the current name.",
            cause: err,
          });
        }
        case "left-at-requested-name": {
          // The anticipated, retryable split state: DB holds the name, only
          // the provider sync is unconfirmed. PRECONDITION_FAILED is
          // classified as expected (warn log, no Sentry). A repair attempt
          // gets its own message since it wrote nothing new.
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: isRepairAttempt
              ? "The workspace name is already saved, but we could not confirm it with our authentication provider. Please retry, or contact support@unkey.com"
              : "The workspace name was saved, but we could not confirm the update with our authentication provider. Please retry the rename to complete it, or contact support@unkey.com",
            cause: err,
          });
        }
        default: {
          // Forces a compile error when a ReconcileOutcome variant is added
          // but not handled above.
          return outcome satisfies never;
        }
      }
    }

    if (isRepairAttempt) {
      await auditRepairSync();
    }
    return { updated: true };
  });
