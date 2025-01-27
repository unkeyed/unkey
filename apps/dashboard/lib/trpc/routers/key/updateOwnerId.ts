import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const updateKeyOwnerId = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      ownerId: z.string().nullish(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.keyId),
            isNull(table.deletedAt),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update ownerId on this key. Please try again or contact support@unkey.dev",
        });
      });
    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            ownerId: input.ownerId ?? null,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update ownerId on this key. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: `Changed ownerId of ${key.id} to ${input.ownerId}`,
          resources: [
            {
              type: "key",
              id: key.id,
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
            "We were unable to update ownerId on this key. Please try again or contact support@unkey.dev",
        });
      });

    return true;
  });
