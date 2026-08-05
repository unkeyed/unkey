import { ActorType } from "@/gen/proto/ctrl/v1/actor_pb";
import { DeployService, DeploymentTrigger } from "@/gen/proto/ctrl/v1/deployment_pb";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { match } from "@unkey/match";
import { z } from "zod";

export const redeploy = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      deploymentId: z.string().min(1, "Deployment ID is required"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ctrl = createCtrlClient(DeployService);

    try {
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
          projectId: true,
          appId: true,
          source: true,
          image: true,
          gitCommitSha: true,
          gitBranch: true,
          gitCommitMessage: true,
          gitCommitAuthorHandle: true,
          gitCommitAuthorAvatarUrl: true,
          gitCommitTimestamp: true,
        },
        with: {
          project: { columns: { id: true, name: true } },
          environment: { columns: { slug: true } },
        },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found or access denied",
        });
      }

      const isGitSourced = await match(deployment.source)
        .with("git_build", () => true)
        .with("docker_image", () => false)
        .with("unknown", async () => {
          const repoConnection = await db.query.githubRepoConnections.findFirst({
            where: (table, { eq }) => eq(table.appId, deployment.appId),
            columns: { appId: true },
          });
          return repoConnection != null;
        })
        .exhaustive();

      const dockerImage = deployment.image;
      if (!isGitSourced && !dockerImage) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Deployment has no resolved Docker image to redeploy",
        });
      }

      const result = await ctrl
        .createDeployment({
          projectId: deployment.project.id,
          appId: deployment.appId,
          environmentSlug: deployment.environment?.slug ?? "",
          trigger: DeploymentTrigger.DASHBOARD,
          triggeredBy: ctx.user.id,
          actor: {
            id: ctx.user.id,
            type: ActorType.USER,
            remoteIp: ctx.audit.location,
            userAgent: ctx.audit.userAgent ?? "",
          },
          ...(isGitSourced
            ? {
                gitCommit: {
                  commitSha: deployment.gitCommitSha ?? "",
                  branch: deployment.gitBranch ?? "",
                  commitMessage: deployment.gitCommitMessage ?? "",
                  authorHandle: deployment.gitCommitAuthorHandle ?? "",
                  authorAvatarUrl: deployment.gitCommitAuthorAvatarUrl ?? "",
                  timestamp: BigInt(deployment.gitCommitTimestamp ?? 0),
                },
              }
            : { dockerImage: dockerImage ?? "" }),
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: err,
          });
        });

      return { deploymentId: result.deploymentId };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Redeploy request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to communicate with control service",
      });
    }
  });
