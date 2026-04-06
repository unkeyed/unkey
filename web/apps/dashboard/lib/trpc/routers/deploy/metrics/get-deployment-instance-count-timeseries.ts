import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { TIME_WINDOWS } from "@unkey/clickhouse";
import { z } from "zod";

export const getDeploymentInstanceCountTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      resourceType: z.enum(["deployment", "sentinel"]),
      resourceId: z.string(),
      instanceName: z.string().optional(),
      window: z.enum(TIME_WINDOWS).default("1h"),
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

    const result = await clickhouse.resources.instances.timeseries({
      workspaceId: ctx.workspace.id,
      resourceType: input.resourceType,
      resourceId: input.resourceId,
      instanceName: input.instanceName ?? "",
      window: input.window,
    });

    if (result.err) {
      console.warn("Failed to fetch instance count timeseries", result.err);
      return [];
    }

    return result.val;
  });
