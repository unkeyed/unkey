import { and, db, eq } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateInstances = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      replicasPerRegion: z.number().min(1).max(10),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const existing = await db.query.environmentRuntimeSettings.findFirst({
      where: and(
        eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
        eq(environmentRuntimeSettings.environmentId, input.environmentId),
      ),
    });

    const currentConfig = (existing?.regionConfig as Record<string, number>) ?? {};
    const currentRegions = Object.keys(currentConfig);

    const regionConfig: Record<string, number> = {};

    if (currentRegions.length > 0) {
      for (const region of currentRegions) {
        regionConfig[region] = input.replicasPerRegion;
      }
    } else {
      const regionsEnv = process.env.AVAILABLE_REGIONS ?? "";
      for (const region of regionsEnv.split(",")) {
        regionConfig[region] = input.replicasPerRegion;
      }
    }

    await db
      .update(environmentRuntimeSettings)
      .set({ regionConfig })
      .where(
        and(
          eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(environmentRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
