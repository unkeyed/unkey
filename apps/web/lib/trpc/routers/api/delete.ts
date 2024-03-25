import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { auth, t } from "../../trpc";

export const deleteApi = t.procedure
  .use(auth)
  .input(
    z.object({
      apiId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const api = await db.query.apis.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.id, input.apiId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!api || api.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.apis)
        .set({ deletedAt: new Date() })
        .where(eq(schema.apis.id, input.apiId));

      await ingestAuditLogs({
        workspaceId: api.workspaceId,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "api.delete",
        description: `Deleted ${api.id}`,
        resources: [
          {
            type: "api",
            id: api.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      }).catch((err) => {
        tx.rollback();
        throw err;
      });

      const keyIds = await tx.query.keys.findMany({
        where: eq(schema.keys.keyAuthId, api.keyAuthId!),
        columns: { id: true },
      });

      if (keyIds.length > 0) {
        await tx
          .update(schema.keys)
          .set({ deletedAt: new Date() })
          .where(eq(schema.keys.keyAuthId, api.keyAuthId!));
        await ingestAuditLogs(
          keyIds.map(({ id }) => ({
            workspaceId: api.workspace.id,
            actor: {
              type: "user",
              id: ctx.user.id,
            },
            event: "key.delete",
            description: `Deleted ${id} as part of the ${api.id} deletion`,
            resources: [
              {
                type: "api",
                id: api.id,
              },
              {
                type: "key",
                id: id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          })),
        ).catch((err) => {
          tx.rollback();
          throw err;
        });
      }
    });
  });
