import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";
import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
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

      // Source type is determined by whether the app has a GitHub repo
      // connection, not by commit metadata on the deployment row — docker
      // redeploys carry forward git metadata from the previous deployment
      // and would otherwise be misclassified as git-sourced.
      const repoConnection = await db.query.githubRepoConnections.findFirst({
        where: (table, { eq }) => eq(table.appId, deployment.appId),
        columns: { appId: true },
      });
      const isGitSourced = repoConnection != null;

      const result = await ctrl
        .createDeployment({
          projectId: deployment.project.id,
          appId: deployment.appId,
          environmentSlug: deployment.environment?.slug ?? "",
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
            : { dockerImage: deployment.image ?? "" }),
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: err,
          });
        });

      await insertAuditLogs(db, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "deployment.redeploy",
        description: `Triggered redeploy for ${deployment.project.name} (deployment ${deployment.id})`,
        resources: [
          {
            type: "deployment",
            id: input.deploymentId,
            name: deployment.project.name,
          },
        ],
        context: ctx.audit,
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
