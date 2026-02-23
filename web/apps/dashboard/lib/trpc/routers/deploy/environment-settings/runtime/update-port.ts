import { and, db, eq } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updatePort = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      port: z.number().int().min(2000).max(54000),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .update(environmentRuntimeSettings)
      .set({ port: input.port })
      .where(
        and(
          eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(environmentRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
