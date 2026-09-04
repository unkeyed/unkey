import { DEPLOYMENT_STATUSES } from "@/lib/collections/deploy/deployment-status";
import { and, db, desc, eq, gte, inArray, lt, lte, or } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { deployments } from "@unkey/db/src/schema";
import { z } from "zod";
import { deploymentListSelect } from "./deployment-query-helpers";
import { enrichDeploymentRows } from "./enrich-deployment-rows";

const MAX_LIMIT = 500;

export const listDeployments = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
      appId: z.string().optional(),
      deploymentIds: z.array(z.string()).min(1).max(100).optional(),
      environmentIds: z.array(z.string()).min(1).max(50).optional(),
      statuses: z.array(z.enum(DEPLOYMENT_STATUSES)).min(1).optional(),
      branches: z.array(z.string()).min(1).max(50).optional(),
      startTime: z.number().int().optional(),
      endTime: z.number().int().optional(),
      limit: z.number().int().min(1).max(MAX_LIMIT).default(100),
      // The last row of the previous page. A keyset rather than a row offset,
      // so a deployment created between two page loads cannot shift a row out
      // of every page. Named `cursor` because that is the page parameter
      // tRPC's useInfiniteQuery injects.
      cursor: z.object({ createdAt: z.number().int(), id: z.string() }).nullish(),
    }),
  )
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    try {
      // One extra row tells the client whether another page exists without a
      // second count query.
      const rows = await db
        .select(deploymentListSelect)
        .from(deployments)
        .where(
          and(
            eq(deployments.workspaceId, ctx.workspace.id),
            eq(deployments.projectId, input.projectId),
            input.appId !== undefined ? eq(deployments.appId, input.appId) : undefined,
            input.deploymentIds ? inArray(deployments.id, input.deploymentIds) : undefined,
            input.environmentIds
              ? inArray(deployments.environmentId, input.environmentIds)
              : undefined,
            input.statuses ? inArray(deployments.status, input.statuses) : undefined,
            input.branches ? inArray(deployments.gitBranch, input.branches) : undefined,
            input.startTime !== undefined ? gte(deployments.createdAt, input.startTime) : undefined,
            input.endTime !== undefined ? lte(deployments.createdAt, input.endTime) : undefined,
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
      const deploymentRows = hasMore ? rows.slice(0, input.limit) : rows;
      const last = deploymentRows.at(-1);
      const nextCursor = hasMore && last ? { createdAt: last.createdAt, id: last.id } : null;

      if (deploymentRows.length === 0) {
        return { deployments: [], nextCursor: null };
      }

      return { deployments: await enrichDeploymentRows(ctx.workspace.id, deploymentRows), nextCursor };
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
