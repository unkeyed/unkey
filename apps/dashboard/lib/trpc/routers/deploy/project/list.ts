import type { Project } from "@/lib/collections/deploy/projects";
import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { deployments, frontlineRoutes, projects } from "@unkey/db/src/schema";

type ProjectRow = {
  id: string;
  name: string;
  slug: string;
  git_repository_url: string | null;
  live_deployment_id: string | null;
  is_rolled_back: boolean;
  git_commit_message: string | null;
  git_branch: string | null;
  git_commit_author_handle: string | null;
  git_commit_author_avatar_url: string | null;
  git_commit_timestamp: number | null;
  domain: string | null;
  latest_deployment_id: string | null;
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
        ${projects.isRolledBack},
        ${deployments.gitCommitMessage},
        ${deployments.gitBranch},
        ${deployments.gitCommitAuthorHandle},
        ${deployments.gitCommitAuthorAvatarUrl},
        ${deployments.gitCommitTimestamp},
        ${frontlineRoutes.fullyQualifiedDomainName},
        (
          SELECT id
          FROM ${deployments} d
          WHERE d.project_id = ${projects.id}
            AND d.workspace_id = ${ctx.workspace.id}
          ORDER BY d.created_at DESC
          LIMIT 1
        ) as latest_deployment_id
      FROM ${projects}
      LEFT JOIN ${deployments}
        ON ${projects.liveDeploymentId} = ${deployments.id}
        AND ${deployments.workspaceId} = ${ctx.workspace.id}
      LEFT JOIN ${frontlineRoutes}
      ON ${projects.id} = ${frontlineRoutes.projectId}
      WHERE ${projects.workspaceId} = ${ctx.workspace.id}
      ORDER BY ${projects.updatedAt} DESC
    `);

    return (result.rows as ProjectRow[]).map(
      (row): Project => ({
        id: row.id,
        name: row.name,
        slug: row.slug,
        gitRepositoryUrl: row.git_repository_url,
        liveDeploymentId: row.live_deployment_id,
        isRolledBack: row.is_rolled_back,
        commitTitle: row.git_commit_message,
        branch: row.git_branch ?? "main",
        author: row.git_commit_author_handle,
        commitTimestamp: Number(row.git_commit_timestamp),
        authorAvatar: row.git_commit_author_avatar_url,
        regions: ["dev:local"],
        domain: row.domain,
        latestDeploymentId: row.latest_deployment_id,
      }),
    );
  });
