import { and, db, eq } from "@/lib/db";
import { appBuildSettings, apps } from "@unkey/db/src/schema";
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
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });
    if (!app) {
      return;
    }
    await db
      .update(appBuildSettings)
      .set({ dockerContext: input.dockerContext })
      .where(
        and(
          eq(appBuildSettings.workspaceId, ctx.workspace.id),
          eq(appBuildSettings.appId, app.id),
          eq(appBuildSettings.environmentId, input.environmentId),
        ),
      );
  });
