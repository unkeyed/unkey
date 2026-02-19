import { and, db, eq } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateCommand = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      command: z.array(z.string()),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .update(environmentRuntimeSettings)
      .set({ command: input.command })
      .where(
        and(
          eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(environmentRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
