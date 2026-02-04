import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getSentinelRps = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      sentinelId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const sentinel = await db.query.sentinels.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.sentinelId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          environmentId: true,
        },
      });

      if (!sentinel) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Sentinel not found",
        });
      }

      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(
            eq(table.environmentId, sentinel.environmentId),
            eq(table.workspaceId, ctx.workspace.id),
          ),
        columns: {
          id: true,
          environmentId: true,
          projectId: true,
        },
      });

      if (!deployment) {
        return undefined;
      }

      try {
        const result = await clickhouse.sentinel.rps.bySentinel({
          workspaceId: ctx.workspace.id,
          deploymentId: deployment.id,
          environmentId: deployment.environmentId,
          projectId: deployment.projectId,
          sentinelId: input.sentinelId,
        });

        if (result.err) {
          console.warn("Failed to fetch sentinel RPS from ClickHouse", result.err);
          return undefined;
        }

        return result.val[0]?.avg_rps;
      } catch (chError) {
        console.warn("Failed to fetch sentinel RPS from ClickHouse", chError);
        return undefined;
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch sentinel RPS",
      });
    }
  });
