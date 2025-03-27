import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const deleteKeys = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      keyIds: z.array(z.string()),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          keys: {
            where: (table, { and, inArray, isNull }) =>
              and(isNull(table.deletedAtM), inArray(table.id, input.keyIds)),
            columns: {
              id: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to delete this key. Please try again or contact support@unkey.dev.",
        });
      });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({ deletedAtM: Date.now() })
          .where(
            and(
              eq(schema.keys.workspaceId, workspace.id),
              inArray(
                schema.keys.id,
                workspace.keys.map((k) => k.id),
              ),
            ),
          );
        insertAuditLogs(
          tx,
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
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "We are unable to delete the key. Please try again or contact support@unkey.dev",
        });
      });
  });
