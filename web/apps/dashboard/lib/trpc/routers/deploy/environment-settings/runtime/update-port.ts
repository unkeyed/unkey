import { and, db, eq } from "@/lib/db";
import { appRuntimeSettings, apps } from "@unkey/db/src/schema";
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
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });
    if (!app) {
      return;
    }
    await db
      .update(appRuntimeSettings)
      .set({ port: input.port })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(appRuntimeSettings.appId, app.id),
          eq(appRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
