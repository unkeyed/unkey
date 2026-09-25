import { and, db, desc, eq, inArray, isNull, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  apis,
  appRuntimeSettings,
  apps,
  deployments,
  frontlineRoutes,
  githubRepoConnections,
  keyAuth,
  projects,
  ratelimitNamespaces,
} from "@unkey/db/src/schema";
import { z } from "zod";

export type OverviewDeployment = {
  id: string;
  appId: string;
  status: (typeof deployments.$inferSelect)["status"];
  branch: string | null;
  commitMessage: string | null;
  commitSha: string | null;
  author: string | null;
  authorAvatar: string | null;
  createdAt: number;
};

export type OverviewApp = {
  id: string;
  name: string;
  slug: string;
  sourceType: "unknown" | "git" | "oci";
  repositoryFullName: string | null;
  domain: string | null;
  isRolledBack: boolean;
  hasCurrentDeployment: boolean;
  latest: OverviewDeployment | null;
};

export type ProjectOverview = {
  project: { id: string; name: string; slug: string };
  apps: OverviewApp[];
  keyspaces: Array<{ apiId: string; keyAuthId: string; name: string; keyCount: number }>;
  ratelimits: Array<{ id: string; name: string }>;
  keyspaceLinks: Array<{ appId: string; keyAuthId: string }>;
};

export const projectOverview = workspaceProcedure
  .input(z.object({ projectId: z.string() }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }): Promise<ProjectOverview> => {
    const workspaceId = ctx.workspace.id;

    const project = await db.query.projects.findFirst({
      where: and(eq(projects.workspaceId, workspaceId), eq(projects.id, input.projectId)),
      columns: { id: true, name: true, slug: true },
    });
    if (!project) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Project not found" });
    }

    const deploymentColumns = {
      id: deployments.id,
      appId: deployments.appId,
      status: deployments.status,
      branch: deployments.gitBranch,
      commitMessage: deployments.gitCommitMessage,
      commitSha: deployments.gitCommitSha,
      author: deployments.gitCommitAuthorHandle,
      authorAvatar: deployments.gitCommitAuthorAvatarUrl,
      createdAt: deployments.createdAt,
    };

    const [appRows, keyspaceRows, ratelimitRows] = await Promise.all([
      db
        .select({
          id: apps.id,
          name: apps.name,
          slug: apps.slug,
          sourceType: apps.sourceType,
          currentDeploymentId: apps.currentDeploymentId,
          isRolledBack: apps.isRolledBack,
        })
        .from(apps)
        .where(and(eq(apps.workspaceId, workspaceId), eq(apps.projectId, project.id)))
        .orderBy(desc(apps.updatedAt), desc(apps.id)),
      db
        .select({
          apiId: apis.id,
          keyAuthId: keyAuth.id,
          name: apis.name,
          keyCount: keyAuth.sizeApprox,
        })
        .from(apis)
        .innerJoin(keyAuth, eq(keyAuth.id, apis.keyAuthId))
        .where(
          and(
            eq(apis.workspaceId, workspaceId),
            eq(apis.projectId, project.id),
            isNull(apis.deletedAtM),
          ),
        )
        .orderBy(desc(keyAuth.sizeApprox)),
      db
        .select({ id: ratelimitNamespaces.id, name: ratelimitNamespaces.name })
        .from(ratelimitNamespaces)
        .where(
          and(
            eq(ratelimitNamespaces.workspaceId, workspaceId),
            eq(ratelimitNamespaces.projectId, project.id),
            isNull(ratelimitNamespaces.deletedAtM),
          ),
        ),
    ]);

    const appIds = appRows.map((a) => a.id);
    const ranked = db
      .select({
        ...deploymentColumns,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${deployments.appId} ORDER BY ${deployments.createdAt} DESC, ${deployments.id} DESC)`.as(
          "rn",
        ),
      })
      .from(deployments)
      .where(and(eq(deployments.workspaceId, workspaceId), eq(deployments.projectId, project.id)))
      .as("ranked");

    const [latestRows, repoRows, routeRows, runtimeRows] = appIds.length
      ? await Promise.all([
          db.select().from(ranked).where(eq(ranked.rn, 1)),
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
          db
            .select({
              appId: frontlineRoutes.appId,
              domain: frontlineRoutes.fullyQualifiedDomainName,
              sticky: frontlineRoutes.sticky,
            })
            .from(frontlineRoutes)
            .where(
              and(
                eq(frontlineRoutes.projectId, project.id),
                inArray(frontlineRoutes.appId, appIds),
              ),
            ),
          db
            .select({
              appId: appRuntimeSettings.appId,
              sentinelConfig: appRuntimeSettings.sentinelConfig,
            })
            .from(appRuntimeSettings)
            .where(
              and(
                eq(appRuntimeSettings.workspaceId, workspaceId),
                inArray(appRuntimeSettings.appId, appIds),
              ),
            ),
        ])
      : [[], [], [], []];

    const toDeployment = ({ rn: _, ...d }: (typeof latestRows)[number]): OverviewDeployment => ({
      ...d,
      createdAt: Number(d.createdAt),
    });
    const latestByApp = new Map(latestRows.map((d) => [d.appId, toDeployment(d)]));
    const repoByApp = new Map(repoRows.map((r) => [r.appId, r.repositoryFullName]));
    const domainByApp = new Map<string, string>();
    for (const r of routeRows) {
      if (!domainByApp.has(r.appId) || r.sticky === "live") {
        domainByApp.set(r.appId, r.domain);
      }
    }

    return {
      project,
      apps: appRows.map((a) => ({
        id: a.id,
        name: a.name,
        slug: a.slug,
        sourceType: a.sourceType,
        repositoryFullName: repoByApp.get(a.id) ?? null,
        domain: domainByApp.get(a.id) ?? null,
        isRolledBack: Boolean(a.isRolledBack),
        hasCurrentDeployment: a.currentDeploymentId != null,
        latest: latestByApp.get(a.id) ?? null,
      })),
      keyspaces: keyspaceRows,
      ratelimits: ratelimitRows,
      keyspaceLinks: keyspaceLinksFrom(runtimeRows, new Set(keyspaceRows.map((k) => k.keyAuthId))),
    };
  });

const sentinelConfigSchema = z.object({
  policies: z
    .array(
      z.object({
        keyauth: z.object({ keySpaceIds: z.array(z.string()) }).optional(),
      }),
    )
    .optional(),
});

function keyspaceLinksFrom(
  rows: Array<{ appId: string; sentinelConfig: unknown }>,
  projectKeyspaces: Set<string>,
): ProjectOverview["keyspaceLinks"] {
  const seen = new Set<string>();
  const links: ProjectOverview["keyspaceLinks"] = [];
  for (const row of rows) {
    const config = parseSentinelConfig(row.sentinelConfig);
    for (const policy of config?.policies ?? []) {
      for (const keyAuthId of policy.keyauth?.keySpaceIds ?? []) {
        const key = `${row.appId}:${keyAuthId}`;
        if (projectKeyspaces.has(keyAuthId) && !seen.has(key)) {
          seen.add(key);
          links.push({ appId: row.appId, keyAuthId });
        }
      }
    }
  }
  return links;
}

function parseSentinelConfig(raw: unknown) {
  const text =
    typeof raw === "string" ? raw : raw instanceof Uint8Array ? Buffer.from(raw).toString() : "";
  try {
    const parsed = sentinelConfigSchema.safeParse(JSON.parse(text));
    return parsed.success ? parsed.data : null;
  } catch {
    return null;
  }
}
