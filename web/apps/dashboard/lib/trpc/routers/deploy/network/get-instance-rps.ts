import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getInstanceRps = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      instanceId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const instance = await db.query.instances.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.instanceId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          deploymentId: true,
        },
      });

      if (!instance) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Instance not found",
        });
      }

      try {
        const result = await clickhouse.sentinel.rps.byInstance({
          workspaceId: ctx.workspace.id,
          deploymentId: instance.deploymentId,
          instanceId: input.instanceId,
        });

        if (result.err) {
          console.warn("Failed to fetch instance RPS from ClickHouse", result.err);
          return undefined;
        }

        return result.val[0]?.avg_rps;
      } catch (chError) {
        console.warn("Failed to fetch instance RPS from ClickHouse", chError);
        return undefined;
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch instance RPS",
      });
    }
  });
