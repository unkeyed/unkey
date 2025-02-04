import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const queryDistinctIdentifiers = rateLimitedProcedure(ratelimit.update)
  .input(z.object({ namespaceId: z.string() }))
  .query(async ({ ctx, input }) => {
    // Get workspace
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          ratelimitNamespaces: {
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),
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
            "Failed to retrieve ratelimit logs unique identifiers due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    if (workspace.ratelimitNamespaces.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Namespace not found",
      });
    }

    const result = await clickhouse.querier.query({
      query: `
          SELECT DISTINCT identifier
          FROM ratelimits.raw_ratelimits_v1
          WHERE workspace_id = {workspaceId: String}
            AND namespace_id = {namespaceId: String}
`,
      schema: z.object({ identifier: z.string() }),
      params: z.object({ workspaceId: z.string(), namespaceId: z.string() }),
    })({
      workspaceId: workspace.id,
      namespaceId: workspace.ratelimitNamespaces[0].id,
    });

    return result.val?.map((i) => i.identifier) ?? [];
  });
