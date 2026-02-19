import { and, db, eq } from "@/lib/db";
import { appRuntimeSettings, apps } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateHealthcheck = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      healthcheck: z
        .object({
          method: z.enum(["GET", "POST"]),
          path: z.string(),
          intervalSeconds: z.number().default(10),
          timeoutSeconds: z.number().default(5),
          failureThreshold: z.number().default(3),
          initialDelaySeconds: z.number().default(0),
        })
        .nullable(),
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
      .set({ healthcheck: input.healthcheck })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(appRuntimeSettings.appId, app.id),
          eq(appRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
