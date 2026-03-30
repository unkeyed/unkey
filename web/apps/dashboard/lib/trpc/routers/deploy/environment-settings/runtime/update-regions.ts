import { and, db, eq, notInArray } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRegionalSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateRegions = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      regionIds: z.array(z.string()).min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const env = await db.query.environments.findFirst({
      where: and(
        eq(environments.id, input.environmentId),
        eq(environments.workspaceId, ctx.workspace.id),
      ),
      columns: { appId: true },
    });
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    const existingSettings = await db.query.appRegionalSettings.findMany({
      where: and(
        eq(appRegionalSettings.workspaceId, ctx.workspace.id),
        eq(appRegionalSettings.environmentId, input.environmentId),
      ),
    });

    const defaultReplicas = existingSettings.at(0)?.replicas ?? 1;
    const defaultPolicyId = existingSettings.at(0)?.horizontalAutoscalingPolicyId ?? null;

    await db
      .delete(appRegionalSettings)
      .where(
        and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          eq(appRegionalSettings.environmentId, input.environmentId),
          notInArray(appRegionalSettings.regionId, input.regionIds),
        ),
      );

    const existingRegionIds = new Set(existingSettings.map((s) => s.regionId));
    const toInsert = input.regionIds
      .filter((regionId) => !existingRegionIds.has(regionId))
      .map((regionId) => ({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        regionId,
        replicas: defaultReplicas,
        horizontalAutoscalingPolicyId: defaultPolicyId,
      }));

    if (toInsert.length > 0) {
      await db.insert(appRegionalSettings).values(toInsert);
    }
  });
