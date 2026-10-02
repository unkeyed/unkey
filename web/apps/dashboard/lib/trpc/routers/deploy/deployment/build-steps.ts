import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { buildStepSchema } from "@unkey/clickhouse/src/build-steps";
import { z } from "zod";

const buildStepResponseSchema = buildStepSchema.omit({ error: true }).extend({
  error: z.string().nullable(),
});

export const getDeploymentBuildSteps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      deploymentId: z.string(),
    }),
  )
  .output(
    z.object({
      steps: z.array(buildStepResponseSchema),
    }),
  )
  .query(async ({ ctx, input }) => {
    // Validate deployment exists and belongs to workspace
    const deployment = await db.query.deployments.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { workspaceId: true, projectId: true },
    });
    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found",
      });
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

    return { steps: stepsResult.val };
  });
