import { and, db, eq, inArray } from "@/lib/db";
import { appRegionalSettings } from "@unkey/db/src/schema";
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

    await db
      .update(appRegionalSettings)
      .set({
        replicas: input.replicasPerRegion,
      })
      .where(
        and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          inArray(appRegionalSettings.environmentId, envIds),
        ),
      );
  });
