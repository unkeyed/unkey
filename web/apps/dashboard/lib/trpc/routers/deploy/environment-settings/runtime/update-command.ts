import { and, db, eq, inArray } from "@/lib/db";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectAppIds } from "../utils";

export const updateCommand = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      command: z.array(z.string()),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const appIds = await resolveProjectAppIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appRuntimeSettings)
      .set({ command: input.command })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          inArray(appRuntimeSettings.appId, appIds),
        ),
      );
  });
