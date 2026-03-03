import { and, db, eq, inArray } from "@/lib/db";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectAppIds } from "../utils";

export const updateMemory = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      memoryMib: z.number(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const appIds = await resolveProjectAppIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appRuntimeSettings)
      .set({ memoryMib: input.memoryMib })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          inArray(appRuntimeSettings.appId, appIds),
        ),
      );
  });
