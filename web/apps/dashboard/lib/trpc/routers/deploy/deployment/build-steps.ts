import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
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

export const getDeploymentBuildSteps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(deploymentLogsRequestSchema)
  .output(deploymentLogsResponseSchema)
  .query(async () => {
    try {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "No deployment steps found for this deployment",
      });
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
