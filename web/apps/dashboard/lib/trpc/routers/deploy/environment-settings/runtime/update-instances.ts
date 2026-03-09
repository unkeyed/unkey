import { and, db, eq } from "@/lib/db";
import { appRegionalSettings } from "@unkey/db/src/schema";
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
    await db
      .update(appRegionalSettings)
      .set({
        replicas: input.replicasPerRegion,
      })
      .where(
        and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          eq(appRegionalSettings.environmentId, input.environmentId),
        ),
      );
  });
