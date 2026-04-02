import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getDeploymentRuntimeLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      deploymentId: z.string(),
      limit: z.number().int().default(50),
    }),
  )
  .output(
    z.object({
      logs: z.array(z.object({ time: z.number(), severity: z.string(), message: z.string() })),
    }),
  )
  .query(async ({ ctx, input }) => {
    const deployment = await db.query.deployments.findFirst({
      where: { id: input.deploymentId, workspaceId: ctx.workspace.id },
    });
    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found",
      });
    }

    const { logsQuery } = await clickhouse.runtimeLogs.logs({
      workspaceId: deployment.workspaceId,
      projectId: deployment.projectId,
      deploymentId: deployment.id,
      environmentId: deployment.environmentId,
      appId: deployment.appId,
      limit: input.limit,
      startTime: deployment.createdAt,
      endTime: Date.now(),
      severity: [],
      message: null,
      cursorTime: null,
    });

    const logsResult = await logsQuery;
    if (logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch runtime logs",
      });
    }

    return {
      logs: logsResult.val.map((log) => ({
        time: log.time,
        severity: log.severity,
        message: log.message,
      })),
    };
  });
