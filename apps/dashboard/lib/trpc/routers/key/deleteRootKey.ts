import { and, db, eq, inArray, isNotNull, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
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
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const rootKeys = await db.query.keys.findMany({
      where: (table, { eq, inArray, isNull, and }) =>
        and(
          eq(table.workspaceId, env().UNKEY_WORKSPACE_ID),
          eq(table.forWorkspaceId, workspace.id),
          inArray(table.id, input.keyIds),
          isNull(table.deletedAt),
        ),
      columns: {
        id: true,
      },
    });

    await db
      .update(schema.keys)
      .set({
        deletedAt: new Date(),
      })
      .where(
        inArray(
          schema.keys.id,
          rootKeys.map((k) => k.id),
        ),
      );

    await ingestAuditLogs(
      rootKeys.map((key) => ({
        workspaceId: workspace.id,
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
  });
