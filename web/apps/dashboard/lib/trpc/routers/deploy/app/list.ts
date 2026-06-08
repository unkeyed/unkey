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
      return [];
    }

    const appIds = appRows.map((a) => a.id);

    // Rank rows per app in SQL (deployments and routes are unbounded, so
    // fetching them all just to keep the first per app doesn't scale). The
    // ORDER BY ends in a unique tiebreaker (id) so ties don't pick
    // nondeterministically.
    const rankedDeployments = db
      .select({
        appId: deployments.appId,
        id: deployments.id,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${deployments.appId} ORDER BY ${deployments.createdAt} DESC, ${deployments.id} DESC)`.as(
          "rn",
        ),
      })
      .from(deployments)
      .where(and(eq(deployments.workspaceId, workspaceId), inArray(deployments.appId, appIds)))
      .as("ranked_deployments");

    const rankedRoutes = db
      .select({
        appId: frontlineRoutes.appId,
        fullyQualifiedDomainName: frontlineRoutes.fullyQualifiedDomainName,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${frontlineRoutes.appId} ORDER BY (${frontlineRoutes.sticky} = 'live') DESC, ${frontlineRoutes.updatedAt} DESC, ${frontlineRoutes.id} DESC)`.as(
          "rn",
        ),
      })
      .from(frontlineRoutes)
      .where(
        and(eq(frontlineRoutes.projectId, input.projectId), inArray(frontlineRoutes.appId, appIds)),
      )
      .as("ranked_routes");

    const currentDeploymentIds = Array.from(
      new Set(appRows.map((a) => a.currentDeploymentId).filter((id): id is string => Boolean(id))),
    );

    const [latestDeploymentRows, routeRows, repoRows, currentDeploymentRows] = await Promise.all([
      db
        .select({ appId: rankedDeployments.appId, id: rankedDeployments.id })
        .from(rankedDeployments)
        .where(eq(rankedDeployments.rn, 1)),
      db
        .select({
          appId: rankedRoutes.appId,
          fullyQualifiedDomainName: rankedRoutes.fullyQualifiedDomainName,
        })
        .from(rankedRoutes)
        .where(eq(rankedRoutes.rn, 1)),
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
      currentDeploymentIds.length
        ? db
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
        : Promise.resolve([]),
    ]);

    const latestDeploymentByApp = new Map(latestDeploymentRows.map((r) => [r.appId, r]));
    const domainByApp = new Map(routeRows.map((r) => [r.appId, r]));
    const repoByApp = new Map(repoRows.map((r) => [r.appId, r]));

    const currentDeploymentById = new Map(currentDeploymentRows.map((d) => [d.id, d]));

    return appRows.map((app): App => {
      const currentDeployment = app.currentDeploymentId
        ? currentDeploymentById.get(app.currentDeploymentId)
        : undefined;
      // Image-based deployments carry no git metadata, so gate on the
      // deployment itself, not on commit fields.
      const hasDeployment = currentDeployment != null;

      return {
        id: app.id,
        projectId: app.projectId,
        name: app.name,
        slug: app.slug,
        defaultBranch: app.defaultBranch,
        currentDeploymentId: app.currentDeploymentId ?? null,
        isRolledBack: Boolean(app.isRolledBack),
        repositoryFullName: repoByApp.get(app.id)?.repositoryFullName ?? null,
        latestDeploymentId: latestDeploymentByApp.get(app.id)?.id ?? null,
        commitTitle: currentDeployment?.gitCommitMessage ?? null,
        branch: currentDeployment?.gitBranch ?? app.defaultBranch,
        author: currentDeployment?.gitCommitAuthorHandle ?? null,
        authorAvatar: currentDeployment?.gitCommitAuthorAvatarUrl ?? null,
        commitTimestamp:
          currentDeployment?.gitCommitTimestamp == null
            ? null
            : Number(currentDeployment.gitCommitTimestamp),
        domain: hasDeployment ? (domainByApp.get(app.id)?.fullyQualifiedDomainName ?? null) : null,
      };
    });
  });
