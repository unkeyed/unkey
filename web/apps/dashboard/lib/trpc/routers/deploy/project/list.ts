import type { Project, ProjectApp } from "@/lib/collections/deploy/projects";
import { and, db, desc, eq, inArray, not, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import {
  apps,
  customDomains,
  deployments,
  environments,
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
        createdAt: projects.createdAt,
      })
      .from(projects)
      .where(and(eq(projects.workspaceId, workspaceId), not(eq(projects.slug, "default"))))
      .orderBy(desc(projects.createdAt));

    if (projectRows.length === 0) {
      return [] satisfies Project[];
    }

    const projectIds = projectRows.map((p) => p.id);

    const rankedDeployments = db
      .select({
        appId: deployments.appId,
        id: deployments.id,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${deployments.appId} ORDER BY (${environments.kind} = 'production') DESC, ${deployments.createdAt} DESC, ${deployments.id} DESC)`.as(
          "rn",
        ),
      })
      .from(deployments)
      .innerJoin(
        environments,
        and(
          eq(environments.id, deployments.environmentId),
          eq(environments.workspaceId, workspaceId),
        ),
      )
      .where(
        and(eq(deployments.workspaceId, workspaceId), inArray(deployments.projectId, projectIds)),
      )
      .as("ranked_deployments");

    const rankedDomains = db
      .select({
        appId: customDomains.appId,
        domain: customDomains.domain,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${customDomains.appId} ORDER BY (${frontlineRoutes.id} IS NOT NULL) DESC, ${customDomains.domain} ASC, ${customDomains.id} ASC)`.as(
          "rn",
        ),
      })
      .from(customDomains)
      .leftJoin(
        frontlineRoutes,
        and(
          eq(frontlineRoutes.appId, customDomains.appId),
          eq(frontlineRoutes.fullyQualifiedDomainName, customDomains.domain),
          eq(frontlineRoutes.sticky, "live"),
        ),
      )
      .innerJoin(
        environments,
        and(
          eq(environments.id, customDomains.environmentId),
          eq(environments.workspaceId, workspaceId),
          eq(environments.kind, "production"),
        ),
      )
      .where(
        and(
          eq(customDomains.workspaceId, workspaceId),
          inArray(customDomains.projectId, projectIds),
          eq(customDomains.verificationStatus, "verified"),
        ),
      )
      .as("ranked_domains");

    const [appRows, headlineDeploymentRows, repoRows, domainRows] = await Promise.all([
      db
        .select({
          id: apps.id,
          projectId: apps.projectId,
          name: apps.name,
          currentDeploymentId: apps.currentDeploymentId,
          updatedAt: apps.updatedAt,
        })
        .from(apps)
        .where(and(eq(apps.workspaceId, workspaceId), inArray(apps.projectId, projectIds)))
        .orderBy(apps.projectId, desc(apps.updatedAt), desc(apps.id)),
      db
        .select({
          appId: rankedDeployments.appId,
          id: deployments.id,
          status: deployments.status,
          gitCommitMessage: deployments.gitCommitMessage,
          gitBranch: deployments.gitBranch,
          createdAt: deployments.createdAt,
        })
        .from(rankedDeployments)
        .innerJoin(
          deployments,
          and(eq(deployments.id, rankedDeployments.id), eq(deployments.workspaceId, workspaceId)),
        )
        .where(eq(rankedDeployments.rn, 1)),
      db
        .select({
          appId: githubRepoConnections.appId,
          repositoryFullName: githubRepoConnections.repositoryFullName,
        })
        .from(githubRepoConnections)
        .where(
          and(
            eq(githubRepoConnections.workspaceId, workspaceId),
            inArray(githubRepoConnections.projectId, projectIds),
          ),
        ),
      db
        .select({ appId: rankedDomains.appId, domain: rankedDomains.domain })
        .from(rankedDomains)
        .where(eq(rankedDomains.rn, 1)),
    ]);

    const headlineDeploymentByApp = new Map<string, ProjectApp["headlineDeployment"]>();
    for (const row of headlineDeploymentRows) {
      headlineDeploymentByApp.set(row.appId, {
        id: row.id,
        status: row.status,
        commitMessage: row.gitCommitMessage ?? null,
        branch: row.gitBranch ?? null,
        deployedAt: Number(row.createdAt),
      });
    }

    const domainByApp = new Map(domainRows.map((row) => [row.appId, row.domain]));

    const repoByApp = new Map(repoRows.map((row) => [row.appId, row.repositoryFullName]));

    const appsByProject = new Map<string, ProjectApp[]>();
    const primaryAppByProject = new Map<string, { appId: string; currentDeploymentId: string }>();
    for (const row of appRows) {
      const list = appsByProject.get(row.projectId) ?? [];
      list.push({
        id: row.id,
        name: row.name,
        customDomain: row.currentDeploymentId ? (domainByApp.get(row.id) ?? null) : null,
        headlineDeployment: headlineDeploymentByApp.get(row.id) ?? null,
      });
      appsByProject.set(row.projectId, list);

      if (row.currentDeploymentId && !primaryAppByProject.has(row.projectId)) {
        primaryAppByProject.set(row.projectId, {
          appId: row.id,
          currentDeploymentId: row.currentDeploymentId,
        });
      }
    }

    return projectRows.map((project): Project => {
      const primaryApp = primaryAppByProject.get(project.id);

      return {
        id: project.id,
        name: project.name,
        slug: project.slug,
        apps: appsByProject.get(project.id) ?? [],
        repositoryFullName: primaryApp ? (repoByApp.get(primaryApp.appId) ?? null) : null,
        currentDeploymentId: primaryApp?.currentDeploymentId ?? null,
        createdAt: Number(project.createdAt),
      };
    });
  });
