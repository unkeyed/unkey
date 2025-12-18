import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const deleteIdentity = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
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
            "We were unable to load this identity. Please try again or contact support@unkey.dev",
        });
      });

    if (!identity) {
      throw new TRPCError({
        message:
          "We are unable to find the correct identity. Please try again or contact support@unkey.dev.",
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
        await tx
          .update(schema.identities)
          .set({
            deleted: true,
          })
          .where(eq(schema.identities.id, identity.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to delete this identity. Please try again or contact support@unkey.dev",
            });
          });

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
            "We are unable to delete this identity. Please try again or contact support@unkey.dev",
        });
      });

    return {
      identityId: identity.id,
      success: true,
    };
  });
