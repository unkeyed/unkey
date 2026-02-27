import type { Project } from "@/lib/collections/deploy/projects";
import { db, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import {
  deployments,
  frontlineRoutes,
  projects,
} from "@unkey/db/src/schema";

type ProjectRow = {
  id: string;
  name: string;
  slug: string;
  repository_full_name: string | null;
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

export const listProjects = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    // Pick the most recently updated app that has a live deployment.
    // This avoids hardcoding slug = 'default' and works for any number of apps.
    const result = await db.execute(sql`
      SELECT
        ${projects.id},
        ${projects.name},
        ${projects.slug},
        ${projects.updatedAt},
        (
          SELECT grc.repository_full_name
          FROM github_repo_connections grc
          WHERE grc.project_id = ${projects.id}
          LIMIT 1
        ) as repository_full_name,
        (
          SELECT a.live_deployment_id
          FROM apps a
          WHERE a.project_id = ${projects.id}
            AND a.live_deployment_id IS NOT NULL
          ORDER BY a.updated_at DESC
          LIMIT 1
        ) as live_deployment_id,
        (
          SELECT a.is_rolled_back
          FROM apps a
          WHERE a.project_id = ${projects.id}
            AND a.live_deployment_id IS NOT NULL
          ORDER BY a.updated_at DESC
          LIMIT 1
        ) as is_rolled_back,
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
        ON ${deployments.id} = (
          SELECT a2.live_deployment_id
          FROM apps a2
          WHERE a2.project_id = ${projects.id}
            AND a2.live_deployment_id IS NOT NULL
          ORDER BY a2.updated_at DESC
          LIMIT 1
        )
        AND ${deployments.workspaceId} = ${ctx.workspace.id}
      LEFT JOIN ${frontlineRoutes}
        ON ${projects.id} = ${frontlineRoutes.projectId}
      WHERE ${projects.workspaceId} = ${ctx.workspace.id}
      ORDER BY ${projects.updatedAt} DESC
    `);

    return (result.rows as ProjectRow[]).map((row): Project => {
      // Single source of truth for "has deployment" in the UI:
      // we consider a deployment present when we have commit metadata from the joined row.
      const hasDeployment = row.git_commit_timestamp !== null;

      return {
        id: row.id,
        name: row.name,
        slug: row.slug,
        repositoryFullName: row.repository_full_name,
        liveDeploymentId: row.live_deployment_id,
        isRolledBack: row.is_rolled_back,
        commitTitle: row.git_commit_message,
        branch: row.git_branch ?? "main",
        author: row.git_commit_author_handle,
        // Preserve null instead of coercing to 0 when there is no deployment
        commitTimestamp:
          row.git_commit_timestamp === null ? null : Number(row.git_commit_timestamp),
        authorAvatar: row.git_commit_author_avatar_url,
        // Only show regions/domain when we have a deployment (and thus commit data)
        regions: hasDeployment ? ["local.dev"] : [],
        domain: hasDeployment ? row.domain : null,
        latestDeploymentId: row.latest_deployment_id,
      };
    });
  });
