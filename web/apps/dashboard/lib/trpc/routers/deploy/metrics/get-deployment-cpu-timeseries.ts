import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentCpuTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      resourceType: z.enum(["deployment", "sentinel"]),
      resourceId: z.string(),
      windowHours: z.number().int().min(1).max(168).default(6),
    }),
  )
  .query(async ({ ctx, input }) => {
    const resource = input.resourceType === "sentinel"
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

    const result = await clickhouse.resources.cpu.timeseries({
      workspaceId: ctx.workspace.id,
      resourceType: input.resourceType,
      resourceId: input.resourceId,
      windowHours: input.windowHours,
    });

    if (result.err) {
      console.warn("Failed to fetch CPU timeseries", result.err);
      return [];
    }

    return result.val;
  });
