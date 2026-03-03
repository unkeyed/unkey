import { and, db, eq, inArray } from "@/lib/db";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectEnvironmentIds } from "../utils";

export const updatePort = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      port: z.number().int().min(2000).max(54000),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const envIds = await resolveProjectEnvironmentIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appRuntimeSettings)
      .set({ port: input.port })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          inArray(appRuntimeSettings.environmentId, envIds),
        ),
      );
  });
