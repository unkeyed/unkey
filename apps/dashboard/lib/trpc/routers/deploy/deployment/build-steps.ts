import { db } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const logEntry = z.object({
  id: z.string(),
  timestamp: z.number(),
  message: z.string(),
});

const deploymentLogsRequestSchema = z.object({
  deploymentId: z.string(),
});

const deploymentLogsResponseSchema = z.object({
  logs: z.array(logEntry),
});

export type DeploymentBuildStep = z.infer<typeof logEntry>;
export type DeploymentRequestSchema = z.infer<typeof deploymentLogsRequestSchema>;

export const getDeploymentBuildSteps = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(deploymentLogsRequestSchema)
  .output(deploymentLogsResponseSchema)
  .query(async ({ input }) => {
    try {
      const steps = await db.query.deploymentSteps.findMany({
        where: (table, { eq, and }) => and(eq(table.deploymentId, input.deploymentId)),
        columns: {
          deploymentId: true,
          workspaceId: true,
          projectId: true,
          status: true,
          message: true,
          createdAt: true,
        },
        orderBy: (table, { asc }) => [asc(table.createdAt)],
      });

      if (steps.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "No deployment steps found for this deployment",
        });
      }

      const logs: DeploymentBuildStep[] = steps.map((step) => ({
        id: `${step.deploymentId}_${step.status}`,
        timestamp: step.createdAt,
        message: step.message,
      }));

      return {
        logs,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment logs",
      });
    }
  });
