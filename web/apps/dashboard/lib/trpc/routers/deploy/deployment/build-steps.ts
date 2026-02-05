import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const buildStepWithLogsSchema = z.object({
  step_id: z.string(),
  started_at: z.number().int(),
  completed_at: z.number().int(),
  name: z.string(),
  cached: z.boolean(),
  error: z.string().nullable(),
  has_logs: z.boolean(),
  logs: z
    .array(
      z.object({
        time: z.number().int(),
        message: z.string(),
      }),
    )
    .optional(),
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
    // Validate deployment exists and belongs to workspace
    const deployment = await db.query.deployments.findFirst({
      where: (table, { eq }) => eq(table.id, input.deploymentId),
    });

    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found",
      });
    }

    if (deployment.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({ code: "FORBIDDEN" });
    }

    // Fetch steps from ClickHouse
    const stepsResult = await clickhouse.buildSteps.getSteps({
      workspaceId: deployment.workspaceId,
      projectId: deployment.projectId,
      deploymentId: input.deploymentId,
    });

    if (stepsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch build steps",
      });
    }

    const steps = stepsResult.val;

    // Optionally fetch logs for steps that have them
    if (input.includeStepLogs && steps.length > 0) {
      const stepIdsWithLogs = steps.filter((s) => s.has_logs).map((s) => s.step_id);

      if (stepIdsWithLogs.length > 0) {
        const logsResult = await clickhouse.buildSteps.getLogs({
          workspaceId: deployment.workspaceId,
          projectId: deployment.projectId,
          deploymentId: input.deploymentId,
          stepIds: stepIdsWithLogs,
          limit: 5000,
        });

        if (!logsResult.err) {
          // Nest logs under each step
          return {
            steps: steps.map((step) => ({
              ...step,
              logs: logsResult.val
                .filter((log) => log.step_id === step.step_id)
                .map((log) => ({ time: log.time, message: log.message })),
            })),
          };
        }
      }
    }

    // Return steps without logs
    return { steps };
  });
