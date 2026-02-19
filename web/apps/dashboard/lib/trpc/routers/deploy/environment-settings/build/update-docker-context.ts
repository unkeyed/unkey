import { and, db, eq } from "@/lib/db";
import { environmentBuildSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateDockerContext = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerContext: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .update(environmentBuildSettings)
      .set({ dockerContext: input.dockerContext })
      .where(
        and(
          eq(environmentBuildSettings.workspaceId, ctx.workspace.id),
          eq(environmentBuildSettings.environmentId, input.environmentId),
        ),
      );
  });
