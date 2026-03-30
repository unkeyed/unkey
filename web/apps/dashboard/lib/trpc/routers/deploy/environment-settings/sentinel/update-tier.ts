import { SENTINEL_TIERS_BY_ID } from "@/lib/constants/sentinel-tiers";
import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const updateSentinelTier = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      sentinelId: z.string(),
      tierId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const tier = SENTINEL_TIERS_BY_ID[input.tierId];
    if (!tier) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Tier not found" });
    }

    const sentinel = await db.query.sentinels.findFirst({
      where: (table, { eq: e, and: a }) =>
        a(e(table.id, input.sentinelId), e(table.workspaceId, ctx.workspace.id)),
    });

    if (!sentinel) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Sentinel not found" });
    }

    await db
      .update(schema.sentinels)
      .set({
        sentinelTierId: tier.id,
        cpuMillicores: tier.cpuMillicores,
        memoryMib: tier.memoryMib,
        version: sentinel.version + 1,
        updatedAt: Date.now(),
      })
      .where(
        and(
          eq(schema.sentinels.id, input.sentinelId),
          eq(schema.sentinels.workspaceId, ctx.workspace.id),
        ),
      );

    return { success: true };
  });
