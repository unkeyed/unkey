import { and, db, eq, inArray } from "@/lib/db";
import { appScalingSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectEnvironmentIds } from "../utils";

export const updateCpu = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      cpuMillicores: z.number(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const envIds = await resolveProjectEnvironmentIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appScalingSettings)
      .set({ cpuMillicores: input.cpuMillicores })
      .where(
        and(
          eq(appScalingSettings.workspaceId, ctx.workspace.id),
          inArray(appScalingSettings.environmentId, envIds),
        ),
      );
  });
