import { and, db, eq } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateRegions = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      regions: z.array(z.string()).min(1),
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
    const regionConfig: Record<string, number> = {};
    for (const region of input.regions) {
      regionConfig[region] = currentConfig[region] ?? 1;
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
