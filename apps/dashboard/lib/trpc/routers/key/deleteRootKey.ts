import { and, db, eq, inArray, schema } from "@/lib/db";
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
      with: {
        keys: {
          where: (table, { and, inArray, isNull }) =>
            and(isNull(table.deletedAt), inArray(table.id, input.keyIds)),
          columns: {
            id: true,
          },
        },
      },
    });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    await db
      .update(schema.keys)
      .set({
        deletedAt: new Date(),
      })
      .where(
        and(
          eq(schema.keys.forWorkspaceId, workspace.id),
          inArray(
            schema.keys.id,
            workspace.keys.map((k) => k.id),
          ),
        ),
      );

    await ingestAuditLogs(
      workspace.keys.map((key) => ({
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
