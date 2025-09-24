import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { Project } from "@/lib/collections/deploy/projects";
import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { deployments, domains, projects } from "@unkey/db/src/schema";

type ProjectRow = {
  id: string;
  name: string;
  slug: string;
  updated_at: number | null;
  git_repository_url: string | null;
  live_deployment_id: string | null;
  rolled_back_deployment_id: string | null;
  git_commit_message: string | null;
  git_branch: string | null;
  git_commit_author_name: string | null;
  git_commit_timestamp: number | null;
  runtime_config: Deployment["runtimeConfig"] | null;
  domain: string | null;
};

export const listProjects = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const result = await db.execute(sql`
      SELECT
        ${projects.id},
        ${projects.name},
        ${projects.slug},
        ${projects.updatedAt},
        ${projects.gitRepositoryUrl},
        ${projects.liveDeploymentId},
        ${projects.rolledBackDeploymentId},
        ${deployments.gitCommitMessage},
        ${deployments.gitBranch},
        ${deployments.gitCommitAuthorName},
        ${deployments.gitCommitTimestamp},
        ${deployments.runtimeConfig},
        ${domains.domain}
      FROM ${projects}
      LEFT JOIN ${deployments}
        ON ${projects.liveDeploymentId} = ${deployments.id}
        AND ${deployments.workspaceId} = ${ctx.workspace.id}
      LEFT JOIN ${domains}
        ON ${projects.id} = ${domains.projectId}
        AND ${domains.workspaceId} = ${ctx.workspace.id}
      WHERE ${projects.workspaceId} = ${ctx.workspace.id}
      ORDER BY ${projects.updatedAt} DESC
    `);

    return (result.rows as ProjectRow[]).map(
      (row): Project => ({
        id: row.id,
        name: row.name,
        slug: row.slug,
        updatedAt: row.updated_at,
        gitRepositoryUrl: row.git_repository_url,
        liveDeploymentId: row.live_deployment_id,
        rolledBackDeploymentId: row.rolled_back_deployment_id,
        commitTitle: row.git_commit_message ?? "[DUMMY] Initial commit",
        branch: row.git_branch ?? "main",
        author: row.git_commit_author_name ?? "[DUMMY] Unknown Author",
        commitTimestamp: row.git_commit_timestamp ?? Date.now() - 86400000,
        regions: row.runtime_config?.regions?.map((r) => r.region) ?? ["us-east-1"],
        domain: row.domain ?? "project-temp.unkey.app",
      }),
    );
  });
