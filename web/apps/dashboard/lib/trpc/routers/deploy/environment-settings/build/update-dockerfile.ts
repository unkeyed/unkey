import { and, db, eq } from "@/lib/db";
import { environmentBuildSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateDockerfile = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerfile: z.string().min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .update(environmentBuildSettings)
      .set({ dockerfile: input.dockerfile })
      .where(
        and(
          eq(environmentBuildSettings.workspaceId, ctx.workspace.id),
          eq(environmentBuildSettings.environmentId, input.environmentId),
        ),
      );
  });
