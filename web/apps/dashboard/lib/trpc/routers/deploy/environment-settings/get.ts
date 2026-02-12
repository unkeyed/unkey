import { and, db, eq } from "@/lib/db";
import { environmentBuildSettings, environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const getEnvironmentSettings = workspaceProcedure
  .input(z.object({ environmentId: z.string() }))
  .query(async ({ ctx, input }) => {
    const [buildSettings, runtimeSettings] = await Promise.all([
      db.query.environmentBuildSettings.findFirst({
        where: and(
          eq(environmentBuildSettings.workspaceId, ctx.workspace.id),
          eq(environmentBuildSettings.environmentId, input.environmentId),
        ),
      }),
      db.query.environmentRuntimeSettings.findFirst({
        where: and(
          eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(environmentRuntimeSettings.environmentId, input.environmentId),
        ),
      }),
    ]);

    return { buildSettings: buildSettings ?? null, runtimeSettings: runtimeSettings ?? null };
  });
