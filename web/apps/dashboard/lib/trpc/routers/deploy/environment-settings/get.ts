import { and, db, eq } from "@/lib/db";
import { environmentBuildSettings, environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";
import type { SentinelConfig } from "./sentinel/update-middleware";

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

    return {
      buildSettings: buildSettings ?? null,
      runtimeSettings: runtimeSettings
        ? {
            ...runtimeSettings,
            // Without that length check this will Buffer.from gives "", and JSON.parse("") throws 500.
            sentinelConfig: runtimeSettings.sentinelConfig?.length
              ? (JSON.parse(
                  Buffer.from(runtimeSettings.sentinelConfig).toString(),
                ) as SentinelConfig)
              : undefined,
          }
        : null,
    };
  });
