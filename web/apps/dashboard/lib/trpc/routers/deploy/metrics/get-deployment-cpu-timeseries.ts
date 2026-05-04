import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { TIME_WINDOWS } from "@unkey/clickhouse";
import { z } from "zod";

export const getDeploymentCpuTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      resourceId: z.string(),
      instanceName: z.string().optional(),
      window: z.enum(TIME_WINDOWS).default("1h"),
    }),
  )
  .query(async ({ ctx, input }) => {
    const resource = await db.query.deployments.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.id, input.resourceId), eq(table.workspaceId, ctx.workspace.id)),
    });

    if (!resource) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Resource not found" });
    }

    const result = await clickhouse.resources.cpu.timeseries({
      workspaceId: ctx.workspace.id,
      resourceId: input.resourceId,
      instanceName: input.instanceName ?? "",
      window: input.window,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch CPU timeseries",
        cause: result.err,
      });
    }

    return result.val;
  });
