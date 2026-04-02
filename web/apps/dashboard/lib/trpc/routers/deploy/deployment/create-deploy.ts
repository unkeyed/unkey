import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";

import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";

import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

import { z } from "zod";

export const createDeploy = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      projectId: z.string().min(1, "Project ID is required"),
      environmentSlug: z.string().min(1, "Environment slug is required"),
      gitRef: z.string().optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ctrl = createCtrlClient(DeployService);

    try {
      const project = await db.query.projects.findFirst({
        where: { id: input.projectId, workspaceId: ctx.workspace.id },
        columns: { id: true, name: true },
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found or access denied",
        });
      }

      // Look up the environment to find the app
      const environment = await db.query.environments.findFirst({
        where: {
          projectId: input.projectId,
          slug: input.environmentSlug,
          workspaceId: ctx.workspace.id,
        },
        columns: { id: true, appId: true },
      });

      if (!environment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Environment '${input.environmentSlug}' not found`,
        });
      }

      const ref = input.gitRef ? parseGitRef(input.gitRef) : undefined;
      // Only full 40-char SHAs are treated as commits; abbreviated SHAs are not supported
      const isCommitSha = (r: string): boolean => /^[0-9a-f]{40}$/i.test(r);

      const result = await ctrl
        .createDeployment({
          projectId: input.projectId,
          appId: environment.appId,
          environmentSlug: input.environmentSlug,
          ...(ref
            ? {
                gitCommit: isCommitSha(ref) ? { commitSha: ref } : { branch: ref },
              }
            : {}),
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
        event: "deployment.create",
        description: `Triggered initial deployment for ${project.name}`,
        resources: [
          {
            type: "deployment",
            id: result.deploymentId,
            name: project.name,
          },
        ],
        context: ctx.audit,
      });

      return { deploymentId: result.deploymentId };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error("Create deployment request failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to communicate with control service",
      });
    }
  });

export function parseGitRef(raw: string): string {
  const trimmed = raw.trim();

  // https://github.com/owner/repo/tree/branch-name (supports slashes in branch)
  const treeMatch = trimmed.match(/^https?:\/\/github\.com\/[^/]+\/[^/]+\/tree\/(.+)$/);
  if (treeMatch) {
    return treeMatch[1];
  }

  // https://github.com/owner/repo/commit/<40-char SHA>
  const commitMatch = trimmed.match(
    /^https?:\/\/github\.com\/[^/]+\/[^/]+\/commit\/([0-9a-f]{40})$/i,
  );
  if (commitMatch) {
    return commitMatch[1];
  }

  return trimmed;
}
