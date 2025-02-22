import { insertAuditLogs } from "@/lib/audit";
import { db, inArray, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const deleteRootKeys = t.procedure
  .use(auth)
  .input(
    z.object({
      keyIds: z.array(z.string()),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const rootKeys = await db.query.keys.findMany({
      where: (table, { eq, inArray, isNull, and }) =>
        and(
          eq(table.workspaceId, env().UNKEY_WORKSPACE_ID),
          eq(table.forWorkspaceId, ctx.workspace.id),
          inArray(table.id, input.keyIds),
          isNull(table.deletedAt),
        ),
      columns: {
        id: true,
      },
    });
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({ deletedAt: new Date(), deletedAtM: Date.now() })
          .where(
            inArray(
              schema.keys.id,
              rootKeys.map((k) => k.id),
            ),
          );
        await insertAuditLogs(
          tx,
          ctx.workspace.auditLogBucket.id,
          rootKeys.map((key) => ({
            workspaceId: ctx.workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "key.delete",
            description: `Deleted ${key.id}`,
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
          })),
        );
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete the root key. Please try again or contact support@unkey.dev",
        });
      });
  });
