import { and, db, eq } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateMemory = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      memoryMib: z.number(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .update(environmentRuntimeSettings)
      .set({ memoryMib: input.memoryMib })
      .where(
        and(
          eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(environmentRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
