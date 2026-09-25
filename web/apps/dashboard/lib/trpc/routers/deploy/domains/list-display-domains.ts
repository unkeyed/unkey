import { and, db, eq, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { customDomains, environments, frontlineRoutes, projects } from "@unkey/db/src/schema";
import { z } from "zod";

export const listDisplayDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;

    const rankedRoutes = db
      .select({
        appId: frontlineRoutes.appId,
        fullyQualifiedDomainName: frontlineRoutes.fullyQualifiedDomainName,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${frontlineRoutes.appId} ORDER BY (${frontlineRoutes.sticky} = 'live') DESC, ${frontlineRoutes.updatedAt} DESC, ${frontlineRoutes.id} DESC)`.as(
          "rn",
        ),
      })
      .from(frontlineRoutes)
      .innerJoin(
        projects,
        and(eq(projects.id, frontlineRoutes.projectId), eq(projects.workspaceId, workspaceId)),
      )
      .where(eq(frontlineRoutes.projectId, input.projectId))
      .as("ranked_routes");

    const rankedCustomDomains = db
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
          eq(customDomains.projectId, input.projectId),
          eq(customDomains.verificationStatus, "verified"),
        ),
      )
      .as("ranked_custom_domains");

    const [routeRows, customDomainRows] = await Promise.all([
      db
        .select({ appId: rankedRoutes.appId, domain: rankedRoutes.fullyQualifiedDomainName })
        .from(rankedRoutes)
        .where(eq(rankedRoutes.rn, 1)),
      db
        .select({ appId: rankedCustomDomains.appId, domain: rankedCustomDomains.domain })
        .from(rankedCustomDomains)
        .where(eq(rankedCustomDomains.rn, 1)),
    ]);

    const customDomainByApp = new Map(customDomainRows.map((row) => [row.appId, row.domain]));
    const appIds = new Set([...routeRows.map((row) => row.appId), ...customDomainByApp.keys()]);
    const domainByApp = new Map(routeRows.map((row) => [row.appId, row.domain]));
    return Array.from(appIds, (appId) => ({
      appId,
      domain: domainByApp.get(appId) ?? null,
      customDomain: customDomainByApp.get(appId) ?? null,
    }));
  });
