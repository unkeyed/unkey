import { and, db, eq } from "@/lib/db";
import { appRuntimeSettings, apps } from "@unkey/db/src/schema";
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
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });
    if (!app) {
      return;
    }
    await db
      .update(appRuntimeSettings)
      .set({ memoryMib: input.memoryMib })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(appRuntimeSettings.appId, app.id),
          eq(appRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
