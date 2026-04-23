import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";

import { DeployService } from "@/gen/proto/ctrl/v1/deployment_pb";

import { and, db, eq } from "@/lib/db";
import { getPullRequest } from "@/lib/github";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { environments, githubRepoConnections } from "@unkey/db/src/schema";
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
        where: (table, { eq, and }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
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
        where: and(
          eq(environments.projectId, input.projectId),
          eq(environments.slug, input.environmentSlug),
          eq(environments.workspaceId, ctx.workspace.id),
        ),
        columns: { id: true, appId: true },
      });

      if (!environment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Environment '${input.environmentSlug}' not found`,
        });
      }

      const parsed = input.gitRef ? parseGitRef(input.gitRef) : undefined;
      const isCommitSha = (r: string): boolean => /^[0-9a-f]{40}$/i.test(r);

      let ref: string | undefined = parsed?.kind === "ref" ? parsed.value : undefined;

      if (parsed?.kind === "pr") {
        const repoConn = await db.query.githubRepoConnections.findFirst({
          where: eq(githubRepoConnections.appId, environment.appId),
          columns: { installationId: true, repositoryFullName: true },
        });
        if (!repoConn) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "No GitHub repository connected to this app",
          });
        }
        const pr = await getPullRequest(
          repoConn.installationId,
          repoConn.repositoryFullName,
          parsed.value,
        );
        const isFork =
          pr.head.repo?.fork === true && pr.head.repo.full_name !== pr.base.repo.full_name;
        const forkOwner = isFork ? pr.head.repo?.full_name.split("/")[0] : undefined;
        ref = forkOwner ? `${forkOwner}:${pr.head.ref}` : pr.head.ref;
      }

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

type GitRefResult = { kind: "pr"; value: number } | { kind: "ref"; value: string };

export function parseGitRef(raw: string): GitRefResult {
  const trimmed = raw.trim();

  const prMatch = trimmed.match(/^https?:\/\/github\.com\/[^/]+\/[^/]+\/pull\/(\d+)\/?$/);
  if (prMatch) {
    return { kind: "pr", value: Number.parseInt(prMatch[1], 10) };
  }

  const treeMatch = trimmed.match(/^https?:\/\/github\.com\/[^/]+\/[^/]+\/tree\/(.+)$/);
  if (treeMatch) {
    return { kind: "ref", value: treeMatch[1] };
  }

  const commitMatch = trimmed.match(
    /^https?:\/\/github\.com\/[^/]+\/[^/]+\/commit\/([0-9a-f]{40})$/i,
  );
  if (commitMatch) {
    return { kind: "ref", value: commitMatch[1] };
  }

  return { kind: "ref", value: trimmed };
}
