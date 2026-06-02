import type { App } from "@/lib/collections/deploy/apps";
import { and, db, desc, eq, inArray, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { apps, deployments, frontlineRoutes, githubRepoConnections } from "@unkey/db/src/schema";
import { z } from "zod";

export const listApps = workspaceProcedure
  .input(z.object({ projectId: z.string() }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }): Promise<App[]> => {
    const workspaceId = ctx.workspace.id;

    const appRows = await db
      .select({
        id: apps.id,
        projectId: apps.projectId,
        name: apps.name,
        slug: apps.slug,
        defaultBranch: apps.defaultBranch,
        currentDeploymentId: apps.currentDeploymentId,
        isRolledBack: apps.isRolledBack,
      })
      .from(apps)
      .where(and(eq(apps.workspaceId, workspaceId), eq(apps.projectId, input.projectId)))
      .orderBy(desc(apps.updatedAt), desc(apps.id));

    if (appRows.length === 0) {
      return [] satisfies App[];
    }

    const appIds = appRows.map((a) => a.id);

    const [latestDeploymentRows, routeRows, repoRows] = await Promise.all([
      // Each "*-by-app" reducer below picks the first row per appId; ORDER BY
      // ends in a unique tiebreaker (id) so ties don't pick nondeterministically.
      db
        .select({
          appId: deployments.appId,
          id: deployments.id,
          createdAt: deployments.createdAt,
        })
        .from(deployments)
        .where(and(eq(deployments.workspaceId, workspaceId), inArray(deployments.appId, appIds)))
        .orderBy(deployments.appId, desc(deployments.createdAt), desc(deployments.id)),
      db
        .select({
          appId: frontlineRoutes.appId,
          fullyQualifiedDomainName: frontlineRoutes.fullyQualifiedDomainName,
          updatedAt: frontlineRoutes.updatedAt,
          id: frontlineRoutes.id,
        })
        .from(frontlineRoutes)
        .where(
          and(
            eq(frontlineRoutes.projectId, input.projectId),
            inArray(frontlineRoutes.appId, appIds),
          ),
        )
        .orderBy(
          frontlineRoutes.appId,
          sql`(${frontlineRoutes.sticky} = 'live') DESC`,
          desc(frontlineRoutes.updatedAt),
          desc(frontlineRoutes.id),
        ),
      db
        .select({
          appId: githubRepoConnections.appId,
          repositoryFullName: githubRepoConnections.repositoryFullName,
        })
        .from(githubRepoConnections)
        .where(
          and(
            eq(githubRepoConnections.workspaceId, workspaceId),
            inArray(githubRepoConnections.appId, appIds),
          ),
        ),
    ]);

    const latestDeploymentByApp = firstByApp(latestDeploymentRows, (row) => row.id);
    const domainByApp = firstByApp(routeRows, (row) => row.fullyQualifiedDomainName);
    const repoByApp = firstByApp(repoRows, (row) => row.repositoryFullName);

    const currentDeploymentIds = Array.from(
      new Set(appRows.map((a) => a.currentDeploymentId).filter((id): id is string => Boolean(id))),
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

    return appRows.map((app): App => {
      const currentDeployment = app.currentDeploymentId
        ? currentDeploymentById.get(app.currentDeploymentId)
        : undefined;
      const hasDeployment = currentDeployment?.gitCommitTimestamp != null;

      return {
        id: app.id,
        projectId: app.projectId,
        name: app.name,
        slug: app.slug,
        defaultBranch: app.defaultBranch,
        currentDeploymentId: app.currentDeploymentId ?? null,
        isRolledBack: Boolean(app.isRolledBack),
        repositoryFullName: repoByApp.get(app.id) ?? null,
        latestDeploymentId: latestDeploymentByApp.get(app.id) ?? null,
        commitTitle: currentDeployment?.gitCommitMessage ?? null,
        branch: currentDeployment?.gitBranch ?? app.defaultBranch,
        author: currentDeployment?.gitCommitAuthorHandle ?? null,
        authorAvatar: currentDeployment?.gitCommitAuthorAvatarUrl ?? null,
        commitTimestamp:
          currentDeployment?.gitCommitTimestamp == null
            ? null
            : Number(currentDeployment.gitCommitTimestamp),
        domain: hasDeployment ? (domainByApp.get(app.id) ?? null) : null,
      };
    });
  });

// Collapse pre-sorted rows to the first value seen per appId.
function firstByApp<T extends { appId: string }>(
  rows: T[],
  value: (row: T) => string,
): Map<string, string> {
  const map = new Map<string, string>();
  for (const row of rows) {
    if (!map.has(row.appId)) {
      map.set(row.appId, value(row));
    }
  }
  return map;
}
