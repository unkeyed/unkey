import { and, db, eq, inArray } from "@/lib/db";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectEnvironmentIds } from "../utils";

export const updateRegions = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      regions: z.array(z.string()).min(1),
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
    const regionConfig: Record<string, number> = {};
    for (const region of input.regions) {
      regionConfig[region] = currentConfig[region] ?? 1;
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
