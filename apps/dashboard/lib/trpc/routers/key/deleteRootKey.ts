import { insertAuditLogs } from "@/lib/audit";
import { db, inArray, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteRootKeys = rateLimitedProcedure(ratelimit.delete)
  .input(
    z.object({
      keyIds: z.array(z.string()),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to delete this root key. Please try again or contact support@unkey.dev.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
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
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({ deletedAt: new Date() })
          .where(
            inArray(
              schema.keys.id,
              rootKeys.map((k) => k.id),
            ),
          );
        await insertAuditLogs(
          tx,
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete the root key. Please try again or contact support@unkey.dev",
        });
      });
  });
