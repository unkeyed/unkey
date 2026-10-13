import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { TIME_WINDOWS } from "@unkey/clickhouse";
import { z } from "zod";
import { requireOwnedDeployment } from "./require-owned-deployment";

export const getDeploymentDiskTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      resourceId: z.string(),
      instanceName: z.string().optional(),
      window: z.enum(TIME_WINDOWS).default("1h"),
    }),
  )
  .query(async ({ ctx, input }) => {
    await requireOwnedDeployment(
      ctx.deploymentOwnership,
      ctx.workspace.id,
      input.resourceId,
      "Resource not found",
    );

    const result = await clickhouse.resources.disk.timeseries({
      workspaceId: ctx.workspace.id,
      resourceId: input.resourceId,
      instanceName: input.instanceName ?? "",
      window: input.window,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch disk timeseries",
        cause: result.err,
      });
    }

    return result.val;
  });
