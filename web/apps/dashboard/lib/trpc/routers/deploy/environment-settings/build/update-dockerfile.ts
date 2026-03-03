import { and, db, eq, inArray } from "@/lib/db";
import { appBuildSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectAppIds } from "../utils";

export const updateDockerfile = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerfile: z.string().min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const appIds = await resolveProjectAppIds(ctx.workspace.id, input.environmentId);

    await db
      .update(appBuildSettings)
      .set({ dockerfile: input.dockerfile })
      .where(
        and(
          eq(appBuildSettings.workspaceId, ctx.workspace.id),
          inArray(appBuildSettings.appId, appIds),
        ),
      );
  });
