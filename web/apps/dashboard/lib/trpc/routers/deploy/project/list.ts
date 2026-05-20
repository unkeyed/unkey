import type { Project } from "@/lib/collections/deploy/projects";
import { and, db, desc, eq, inArray, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import {
  apps,
  deployments,
  frontlineRoutes,
  githubRepoConnections,
  projects,
} from "@unkey/db/src/schema";

export const listProjects = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const workspaceId = ctx.workspace.id;

    const projectRows = await db
      .select({
        id: projects.id,
        name: projects.name,
        slug: projects.slug,
        updatedAt: projects.updatedAt,
      })
      .from(projects)
      .where(eq(projects.workspaceId, workspaceId))
      .orderBy(desc(projects.updatedAt));

    if (projectRows.length === 0) {
      return [] satisfies Project[];
    }

    const projectIds = projectRows.map((p) => p.id);

    const [appRows, latestDeploymentRows, routeRows, repoRows] = await Promise.all([
      // Each "*-by-project" query below relies on the reducer below picking
      // the first row per projectId. Every ORDER BY therefore ends in a unique
      // tiebreaker (id) so ties on updatedAt/createdAt don't produce a
      // nondeterministic winner across requests.
      db
        .select({
          projectId: apps.projectId,
          currentDeploymentId: apps.currentDeploymentId,
          isRolledBack: apps.isRolledBack,
          updatedAt: apps.updatedAt,
        })
        .from(apps)
        .where(
          and(
            eq(apps.workspaceId, workspaceId),
            inArray(apps.projectId, projectIds),
            sql`${apps.currentDeploymentId} IS NOT NULL`,
          ),
        )
        .orderBy(apps.projectId, desc(apps.updatedAt), desc(apps.id)),
      db
        .select({
          projectId: deployments.projectId,
          id: deployments.id,
          createdAt: deployments.createdAt,
        })
        .from(deployments)
        .where(
          and(eq(deployments.workspaceId, workspaceId), inArray(deployments.projectId, projectIds)),
        )
        .orderBy(deployments.projectId, desc(deployments.createdAt), desc(deployments.id)),
      db
        .select({
          projectId: frontlineRoutes.projectId,
          fullyQualifiedDomainName: frontlineRoutes.fullyQualifiedDomainName,
          isLive: sql<number>`(${frontlineRoutes.sticky} = 'live')`,
          updatedAt: frontlineRoutes.updatedAt,
          id: frontlineRoutes.id,
        })
        .from(frontlineRoutes)
        // frontline_routes has no workspace_id column, so join to projects to
        // enforce workspace ownership explicitly. projectIds is already
        // workspace-scoped, but the policy is that every query with user input
        // must include ctx.workspace.id in its WHERE/JOIN conditions.
        .innerJoin(
          projects,
          and(eq(projects.id, frontlineRoutes.projectId), eq(projects.workspaceId, workspaceId)),
        )
        .where(inArray(frontlineRoutes.projectId, projectIds))
        .orderBy(
          frontlineRoutes.projectId,
          sql`(${frontlineRoutes.sticky} = 'live') DESC`,
          desc(frontlineRoutes.updatedAt),
          desc(frontlineRoutes.id),
        ),
      db
        .select({
          projectId: githubRepoConnections.projectId,
          repositoryFullName: githubRepoConnections.repositoryFullName,
        })
        .from(githubRepoConnections)
        .where(eq(githubRepoConnections.workspaceId, workspaceId)),
    ]);

    const primaryAppByProject = new Map<
      string,
      { currentDeploymentId: string; isRolledBack: boolean }
    >();
    for (const row of appRows) {
      if (!row.currentDeploymentId || primaryAppByProject.has(row.projectId)) {
        continue;
      }
      primaryAppByProject.set(row.projectId, {
        currentDeploymentId: row.currentDeploymentId,
        isRolledBack: Boolean(row.isRolledBack),
      });
    }

    const latestDeploymentByProject = new Map<string, string>();
    for (const row of latestDeploymentRows) {
      if (latestDeploymentByProject.has(row.projectId)) {
        continue;
      }
      latestDeploymentByProject.set(row.projectId, row.id);
    }

    const domainByProject = new Map<string, string>();
    for (const row of routeRows) {
      if (domainByProject.has(row.projectId)) {
        continue;
      }
      domainByProject.set(row.projectId, row.fullyQualifiedDomainName);
    }

    const repoByProject = new Map<string, string>();
    for (const row of repoRows) {
      if (repoByProject.has(row.projectId)) {
        continue;
      }
      repoByProject.set(row.projectId, row.repositoryFullName);
    }

    const currentDeploymentIds = Array.from(
      new Set(
        Array.from(primaryAppByProject.values())
          .map((app) => app.currentDeploymentId)
          .filter((id): id is string => Boolean(id)),
      ),
    );

    const currentDeploymentRows = currentDeploymentIds.length
      ? await db
          .select({
            id: deployments.id,
            gitCommitMessage: deployments.gitCommitMessage,
            gitBranch: deployments.gitBranch,
            gitCommitAuthorHandle: deployments.gitCommitAuthorHandle,
            gitCommitAuthorAvatarUrl: deployments.gitCommitAuthorAvatarUrl,
            gitCommitTimestamp: deployments.gitCommitTimestamp,
          })
          .from(deployments)
          .where(
            and(
              eq(deployments.workspaceId, workspaceId),
              inArray(deployments.id, currentDeploymentIds),
            ),
          )
      : [];

    const currentDeploymentById = new Map(currentDeploymentRows.map((d) => [d.id, d]));

    return projectRows.map((project): Project => {
      const primaryApp = primaryAppByProject.get(project.id);
      const currentDeployment = primaryApp
        ? currentDeploymentById.get(primaryApp.currentDeploymentId)
        : undefined;
      const hasDeployment = currentDeployment?.gitCommitTimestamp != null;

      return {
        id: project.id,
        name: project.name,
        slug: project.slug,
        repositoryFullName: repoByProject.get(project.id) ?? null,
        currentDeploymentId: primaryApp?.currentDeploymentId ?? null,
        isRolledBack: primaryApp?.isRolledBack ?? false,
        commitTitle: currentDeployment?.gitCommitMessage ?? null,
        branch: currentDeployment?.gitBranch ?? "main",
        author: currentDeployment?.gitCommitAuthorHandle ?? null,
        authorAvatar: currentDeployment?.gitCommitAuthorAvatarUrl ?? null,
        commitTimestamp:
          currentDeployment?.gitCommitTimestamp == null
            ? null
            : Number(currentDeployment.gitCommitTimestamp),
        domain: hasDeployment ? (domainByProject.get(project.id) ?? null) : null,
        latestDeploymentId: latestDeploymentByProject.get(project.id) ?? null,
      };
    });
  });
