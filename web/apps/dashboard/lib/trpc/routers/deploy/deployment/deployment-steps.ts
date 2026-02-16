import { db } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { deploymentSteps } from "@unkey/db/src/schema";
import { z } from "zod";

const stepSchema = z.object({
  startedAt: z.number(),
  endedAt: z.number().nullable(),
  duration: z.number().nullable(),
  completed: z.boolean(),
  error: z.string().nullable(),
});

export const getDeploymentSteps = workspaceProcedure
  .input(z.object({ deploymentId: z.string() }))
  .output(z.partialRecord(z.enum(deploymentSteps.step.enumValues), stepSchema.nullable()))
  .query(async ({ ctx, input }) => {
    const deployment = await db.query.deployments.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { id: true },
      with: {
        steps: {
          columns: {
            step: true,
            startedAt: true,
            endedAt: true,
            error: true,
          },
        },
      },
    });

    if (!deployment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Deployment not found",
      });
    }

    const result: Record<string, z.infer<typeof stepSchema>> = {};

    for (const step of deployment.steps) {
      result[step.step] = {
        startedAt: step.startedAt,
        endedAt: step.endedAt ?? null,
        duration: step.endedAt ? step.endedAt - step.startedAt : null,
        completed: Boolean(step.endedAt && !step.error),
        error: step.error ?? null,
      };
    }

    return result;
  });
