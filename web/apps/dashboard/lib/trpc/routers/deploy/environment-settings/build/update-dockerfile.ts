import { and, db, eq } from "@/lib/db";
import { appBuildSettings, apps } from "@unkey/db/src/schema";
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
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });
    if (!app) {
      return;
    }
    await db
      .update(appBuildSettings)
      .set({ dockerfile: input.dockerfile })
      .where(
        and(
          eq(appBuildSettings.workspaceId, ctx.workspace.id),
          eq(appBuildSettings.appId, app.id),
          eq(appBuildSettings.environmentId, input.environmentId),
        ),
      );
  });
