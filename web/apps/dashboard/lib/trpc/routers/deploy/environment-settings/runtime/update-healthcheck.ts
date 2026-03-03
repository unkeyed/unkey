import { and, db, eq, inArray } from "@/lib/db";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectAppIds } from "../utils";

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
    const appIds = await resolveProjectAppIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appRuntimeSettings)
      .set({ healthcheck: input.healthcheck })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          inArray(appRuntimeSettings.appId, appIds),
        ),
      );
  });
