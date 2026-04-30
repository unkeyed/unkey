import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentResourceSummary = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      resourceType: z.enum(["deployment", "sentinel"]),
      resourceId: z.string(),
      instanceName: z.string().optional(),
    }),
  )
  .query(async ({ ctx, input }) => {
    const resource =
      input.resourceType === "sentinel"
        ? await db.query.sentinels.findFirst({
            where: (table, { eq, and }) =>
              and(eq(table.id, input.resourceId), eq(table.workspaceId, ctx.workspace.id)),
          })
        : await db.query.deployments.findFirst({
            where: (table, { eq, and }) =>
              and(eq(table.id, input.resourceId), eq(table.workspaceId, ctx.workspace.id)),
          });

    if (!resource) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Resource not found" });
    }

    const result = await clickhouse.resources.summary({
      workspaceId: ctx.workspace.id,
      resourceType: input.resourceType,
      resourceId: input.resourceId,
      instanceName: input.instanceName ?? "",
    });

    if (result.err) {
      // Bubble a typed error so the UI can render a degraded state
      // instead of treating a ClickHouse outage as "no data". The
      // resource-existence check above already returns NOT_FOUND for
      // truly missing resources, so reaching here means a real failure.
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch resource summary",
        cause: result.err,
      });
    }

    return result.val[0] ?? null;
  });
