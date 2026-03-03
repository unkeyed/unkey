import { and, db, eq, inArray } from "@/lib/db";
import { appBuildSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectEnvironmentIds } from "../utils";

export const updateDockerContext = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerContext: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const envIds = await resolveProjectEnvironmentIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appBuildSettings)
      .set({ dockerContext: input.dockerContext })
      .where(
        and(
          eq(appBuildSettings.workspaceId, ctx.workspace.id),
          inArray(appBuildSettings.environmentId, envIds),
        ),
      );
  });
