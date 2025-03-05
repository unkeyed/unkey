import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const updateKeyEnabled = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.keyId), isNull(table.deletedAtM)),
        with: {
          workspace: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update enabled on this key. Please try again or contact support@unkey.dev",
        });
      });
    if (!key || key.workspace.orgId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the the correct key. Please try again or contact support@unkey.dev.",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            enabled: input.enabled,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update enabled on this key. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
          workspaceId: key.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: `${input.enabled ? "Enabled" : "Disabled"} ${key.id}`,
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
            "We were unable to update enabled on this key. Please try again or contact support@unkey.dev",
        });
      });
  });
