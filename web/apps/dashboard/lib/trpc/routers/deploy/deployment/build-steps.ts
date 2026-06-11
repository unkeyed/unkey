import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { buildStepLogSchema, buildStepSchema } from "@unkey/clickhouse/src/build-steps";
import { z } from "zod";

const buildStepWithLogsSchema = buildStepSchema.omit({ error: true }).extend({
  error: z.string().nullable(),
  logs: z.array(buildStepLogSchema.pick({ time: true, message: true })).optional(),
});

export const getDeploymentBuildSteps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      deploymentId: z.string(),
      includeStepLogs: z.boolean().default(false),
    }),
  )
  .output(
    z.object({
      steps: z.array(buildStepWithLogsSchema),
    }),
  )
  .query(async ({ ctx, input }) => {
    const deployment = await db.query.deployments.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
    });
    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found",
      });
    }

    // One CH round trip instead of two. The combined query LEFT JOINs the
    // logs table to the steps table and returns logs as parallel arrays
    // per step; logLimit=1 keeps the CH-side scan tiny when the caller
    // doesn't want logs (we discard the arrays below).
    const result = await clickhouse.buildSteps.getStepsWithLogs({
      workspaceId: deployment.workspaceId,
      projectId: deployment.projectId,
      deploymentId: input.deploymentId,
      logLimit: input.includeStepLogs ? 20 : 1,
    });
    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch build steps",
      });
    }

    return {
      steps: result.val.map((row) => ({
        step_id: row.step_id,
        started_at: row.started_at,
        completed_at: row.completed_at,
        name: row.name,
        cached: row.cached,
        error: row.error,
        has_logs: row.has_logs,
        logs: input.includeStepLogs
          ? row.log_times.map((time, i) => ({ time, message: row.log_messages[i] ?? "" }))
          : undefined,
      })),
    };
  });
