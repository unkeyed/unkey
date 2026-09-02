import { deployments, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { and, db, desc, eq, isNotNull, lt, ne, or, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { deploymentListSelect, excludeSkipped } from "./deployment-query-helpers";
import { enrichDeploymentRows } from "./enrich-deployment-rows";

const MAX_LIMIT = 100;

export const listActiveBranches = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
      appId: z.string(),
      limit: z.number().int().min(1).max(MAX_LIMIT).default(10),
      cursor: z.object({ createdAt: z.number().int(), id: z.string() }).nullish(),
    }),
  )
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
          ne(deployments.source, "oci"),
          isNotNull(deployments.gitBranch),
          ne(deployments.gitBranch, ""),
          // Filtered before the row number is assigned, so a branch whose most
          // recent push was skipped still shows the deployment actually live on
          // it instead of dropping off the list entirely.
          excludeSkipped(),
        ),
      )
      .as("ranked");

    const rows = await db
      .select(deploymentListSelect)
      .from(deployments)
      .innerJoin(ranked, eq(ranked.id, deployments.id))
      .innerJoin(environments, eq(environments.id, deployments.environmentId))
      .where(
        and(
          eq(ranked.rn, 1),
          eq(environments.kind, "preview"),
          input.cursor
            ? or(
                lt(deployments.createdAt, input.cursor.createdAt),
                and(
                  eq(deployments.createdAt, input.cursor.createdAt),
                  lt(deployments.id, input.cursor.id),
                ),
              )
            : undefined,
        ),
      )
      .orderBy(desc(deployments.createdAt), desc(deployments.id))
      .limit(input.limit + 1);

    const hasMore = rows.length > input.limit;
    const branchRows = hasMore ? rows.slice(0, input.limit) : rows;
    const last = branchRows.at(-1);
    const nextCursor = hasMore && last ? { createdAt: last.createdAt, id: last.id } : null;

    if (branchRows.length === 0) {
      return { branches: [], nextCursor: null };
    }

    return {
      branches: await enrichDeploymentRows(ctx.workspace.id, branchRows),
      nextCursor,
    };
  });
