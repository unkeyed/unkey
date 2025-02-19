import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const searchNamespace = rateLimitedProcedure(ratelimit.update)
  .input(z.object({ query: z.string() }))
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
            "Failed to verify workspace access. Please try again or contact support@unkey.dev if this persists.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const escapedQuery = input.query.replace(/[%_]/g, "\\$&");

    return await db.query.ratelimitNamespaces.findMany({
      where: (table, { isNull, and, like, eq }) =>
        and(
          eq(table.workspaceId, workspace.id),
          like(table.name, `%${escapedQuery}%`),
          isNull(table.deletedAt),
        ),
      columns: {
        id: true,
        name: true,
      },
    });
  });
