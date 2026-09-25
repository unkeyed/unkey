import { and, db, eq, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { frontlineRoutes, projects } from "@unkey/db/src/schema";
import { z } from "zod";

export const listDisplayDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
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
        and(eq(projects.id, frontlineRoutes.projectId), eq(projects.workspaceId, ctx.workspace.id)),
      )
      .where(eq(frontlineRoutes.projectId, input.projectId))
      .as("ranked_routes");

    return db
      .select({ appId: rankedRoutes.appId, domain: rankedRoutes.fullyQualifiedDomainName })
      .from(rankedRoutes)
      .where(eq(rankedRoutes.rn, 1));
  });
