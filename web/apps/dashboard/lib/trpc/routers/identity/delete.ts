import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { isDuplicateKeyError } from "@/lib/utils/db-errors";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const deleteIdentity = workspaceProcedure
  .input(
    z.object({
      identityId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const identity = await db.query.identities
      .findFirst({
        where: (table, { eq, and }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.identityId),
            eq(table.deleted, false),
          ),
      })
      .catch((err) => {
        console.error("Failed to fetch identity:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to load this identity. Please try again or contact support@unkey.com",
        });
      });

    if (!identity) {
      throw new TRPCError({
        message:
          "We are unable to find the correct identity. Please try again or contact support@unkey.com.",
        code: "NOT_FOUND",
      });
    }

    if (identity.deleted) {
      throw new TRPCError({
        message: "This identity has already been deleted.",
        code: "BAD_REQUEST",
      });
    }

    await db
      .transaction(async (tx) => {
        const softDelete = () =>
          tx
            .update(schema.identities)
            .set({ deleted: true })
            .where(eq(schema.identities.id, identity.id));

        // True if this identity is already soft-deleted, in which case a
        // concurrent request beat us to it and the delete is idempotently done.
        const isAlreadySoftDeleted = async () =>
          Boolean(
            await tx.query.identities.findFirst({
              where: (table, { and, eq }) =>
                and(eq(table.id, identity.id), eq(table.deleted, true)),
              columns: { id: true },
            }),
          );

        try {
          await softDelete();
        } catch (err) {
          // The unique index spans (workspaceId, externalId, deleted), so at
          // most one soft-deleted row may exist per externalId. A collision
          // here means either a stale deleted=true row left over from a
          // previously deleted+recreated externalId, or a concurrent delete of
          // this same identity. Mirrors the public API handler
          // (svc/api/routes/v2_identities_delete_identity/handler.go).
          if (!isDuplicateKeyError(err)) {
            throw err;
          }

          const stale = await tx.query.identities.findFirst({
            where: (table, { and, eq, ne }) =>
              and(
                eq(table.workspaceId, ctx.workspace.id),
                eq(table.externalId, identity.externalId),
                eq(table.deleted, true),
                ne(table.id, identity.id),
              ),
            columns: { id: true },
          });

          if (!stale) {
            // No stale row to clear: a concurrent request already soft-deleted
            // this identity. Treat as an idempotent success and skip audit
            // logs — the concurrent request already wrote them.
            if (await isAlreadySoftDeleted()) {
              return;
            }
            throw err;
          }

          // Hard-delete the stale row (and its ratelimits), then retry.
          await tx.delete(schema.ratelimits).where(eq(schema.ratelimits.identityId, stale.id));
          await tx.delete(schema.identities).where(eq(schema.identities.id, stale.id));

          try {
            await softDelete();
          } catch (retryErr) {
            // A concurrent request soft-deleted this identity between our
            // cleanup and retry. Idempotent success — audit logs already
            // written by that request.
            if (isDuplicateKeyError(retryErr) && (await isAlreadySoftDeleted())) {
              return;
            }
            throw retryErr;
          }
        }

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "identity.delete",
          description: `Deleted identity ${identity.id}`,
          resources: [
            {
              type: "identity",
              id: identity.id,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete this identity. Please try again or contact support@unkey.com",
        });
      });

    return {
      identityId: identity.id,
      success: true,
    };
  });
