import { and, db, eq, inArray } from "@/lib/db";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectEnvironmentIds } from "../utils";

export const updateInstances = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      replicasPerRegion: z.number().min(1).max(10),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const envIds = await resolveProjectEnvironmentIds(ctx.workspace.id, input.environmentId);

    const existing = await db.query.appRuntimeSettings.findFirst({
      where: and(
        eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
        eq(appRuntimeSettings.environmentId, input.environmentId),
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
      .update(appRuntimeSettings)
      .set({ regionConfig })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          inArray(appRuntimeSettings.environmentId, envIds),
        ),
      );
  });
