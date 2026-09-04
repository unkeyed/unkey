import { and, db, desc, eq, isNotNull, ne, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { deployments, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { deploymentListSelect } from "./deployment-query-helpers";
import { enrichDeploymentRows } from "./enrich-deployment-rows";

const ACTIVE_BRANCHES_LIMIT = 100;

export const listActiveBranches = workspaceProcedure
  .input(z.object({ projectId: z.string(), appId: z.string() }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    const ranked = db
      .select({
        id: deployments.id,
        rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${deployments.gitBranch} ORDER BY ${deployments.createdAt} DESC, ${deployments.id} DESC)`.as(
          "rn",
        ),
      })
      .from(deployments)
      .where(
        and(
          eq(deployments.workspaceId, ctx.workspace.id),
          eq(deployments.projectId, input.projectId),
          eq(deployments.appId, input.appId),
          isNotNull(deployments.gitBranch),
          ne(deployments.gitBranch, ""),
        ),
      )
      .as("ranked");

    const rows = await db
      .select(deploymentListSelect)
      .from(deployments)
      .innerJoin(ranked, eq(ranked.id, deployments.id))
      .innerJoin(environments, eq(environments.id, deployments.environmentId))
      .where(and(eq(ranked.rn, 1), eq(environments.kind, "preview")))
      .orderBy(desc(deployments.createdAt), desc(deployments.id))
      .limit(ACTIVE_BRANCHES_LIMIT);

    return enrichDeploymentRows(ctx.workspace.id, rows);
  });
