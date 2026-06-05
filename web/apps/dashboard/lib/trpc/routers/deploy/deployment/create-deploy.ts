import { insertAuditLogs } from "@/lib/audit";
import { createCtrlClient } from "@/lib/ctrl-client";

import { DeployService, DeploymentTrigger } from "@/gen/proto/ctrl/v1/deployment_pb";

import { and, db, eq } from "@/lib/db";
import { getPullRequest, listPullRequestsForCommit } from "@/lib/github";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { environments, githubRepoConnections } from "@unkey/db/src/schema";
import { match } from "@unkey/match";
import { z } from "zod";
import {
  type DeployRef,
  detectForkRepo,
  parseDeployRef,
  resolveSourceRepo,
} from "./resolve-deploy-ref";

type RepoConn = { installationId: number; repositoryFullName: string };

type GitCommit = {
  commitSha?: string;
  branch?: string;
  forkRepository: string;
};

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

      let gitCommit: GitCommit | undefined;
      if (input.gitRef) {
        const parsed = parseDeployRef(input.gitRef);

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

        gitCommit = await resolveDeployRef(parsed, repoConn);
      }

      const result = await ctrl
        .createDeployment({
          projectId: input.projectId,
          appId: environment.appId,
          environmentSlug: input.environmentSlug,
          trigger: DeploymentTrigger.DASHBOARD,
          triggeredBy: ctx.user.id,
          ...(gitCommit ? { gitCommit } : {}),
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

function validateSourceRepo(sourceRepo: string, repoConn: RepoConn): string {
  const fork = resolveSourceRepo(sourceRepo, repoConn.repositoryFullName);
  if (!fork && sourceRepo.includes("/") && sourceRepo !== repoConn.repositoryFullName) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: `Repository "${sourceRepo}" is not a fork of "${repoConn.repositoryFullName}"`,
    });
  }
  return fork ?? "";
}

async function resolveFork(
  parsed: Extract<DeployRef, { kind: "sha" } | { kind: "branch" }>,
  repoConn: RepoConn,
): Promise<string> {
  if (parsed.sourceRepo) {
    return validateSourceRepo(parsed.sourceRepo, repoConn);
  }

  return match(parsed)
    .with({ kind: "sha" }, async (ref) => {
      try {
        const prs = await listPullRequestsForCommit(
          repoConn.installationId,
          repoConn.repositoryFullName,
          ref.sha,
        );
        for (const pr of prs) {
          const fork = detectForkRepo(pr);
          if (fork) {
            return fork;
          }
        }
      } catch {
        // Commit may not be associated with any PR
      }
      return "";
    })
    .with({ kind: "branch" }, () => "")
    .exhaustive();
}

async function resolveDeployRef(parsed: DeployRef, repoConn: RepoConn): Promise<GitCommit> {
  return match(parsed)
    .with({ kind: "pr" }, async (ref) => {
      // Fetch the PR through the connected repo's installation auth.
      // GitHub will only return a PR that actually exists on
      // repoConn.repositoryFullName, so a successful fetch is itself
      // proof that the PR base matches the connected repo. We then
      // verify that the URL's parsed source repo matches the PR head
      // GitHub reports, which catches typo-squat URLs like
      // `https://github.com/evil/myapp/pull/123` aimed at a connected
      // `unkey/myapp`. Splitting on `/` and comparing the repo name —
      // the previous behavior — would let those through.
      let pr: Awaited<ReturnType<typeof getPullRequest>>;
      try {
        pr = await getPullRequest(
          repoConn.installationId,
          repoConn.repositoryFullName,
          ref.prNumber,
        );
      } catch (err) {
        console.error("Failed to fetch pull request", { prNumber: ref.prNumber, err });
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: `Failed to fetch pull request #${ref.prNumber}`,
          cause: err,
        });
      }

      const headRepo = pr.head.repo?.full_name;
      // GitHub owner/repo paths are case-insensitive, so don't reject a
      // user-supplied URL just because of casing drift.
      if (!headRepo || headRepo.toLowerCase() !== ref.sourceRepo.toLowerCase()) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Pull request URL points to "${ref.sourceRepo}", but PR #${ref.prNumber} on "${repoConn.repositoryFullName}" is from "${headRepo ?? "unknown"}"`,
        });
      }

      return {
        branch: pr.head.ref,
        forkRepository: detectForkRepo(pr) ?? "",
      };
    })
    .with({ kind: "sha" }, async (ref) => {
      const forkRepository = await resolveFork(ref, repoConn);
      return { commitSha: ref.sha, forkRepository };
    })
    .with({ kind: "branch" }, async (ref) => {
      const forkRepository = await resolveFork(ref, repoConn);
      return { branch: ref.branch, forkRepository };
    })
    .exhaustive();
}
