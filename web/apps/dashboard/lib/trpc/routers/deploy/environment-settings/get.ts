import type { Config } from "@/gen/proto/config/v1/config_pb";
import { and, db, eq } from "@/lib/db";
import { appBuildSettings, appRuntimeSettings, apps } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const getEnvironmentSettings = workspaceProcedure
  .input(z.object({ environmentId: z.string() }))
  .query(async ({ ctx, input }) => {
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });

    if (!app) {
      return { buildSettings: null, runtimeSettings: null };
    }

    const [buildSettings, runtimeSettings] = await Promise.all([
      db.query.appBuildSettings.findFirst({
        where: and(
          eq(appBuildSettings.workspaceId, ctx.workspace.id),
          eq(appBuildSettings.appId, app.id),
          eq(appBuildSettings.environmentId, input.environmentId),
        ),
      }),
      db.query.appRuntimeSettings.findFirst({
        where: and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(appRuntimeSettings.appId, app.id),
          eq(appRuntimeSettings.environmentId, input.environmentId),
        ),
      }),
    ]);

    return {
      buildSettings: buildSettings ?? null,
      runtimeSettings: runtimeSettings
        ? {
            ...runtimeSettings,
            sentinelConfig: runtimeSettings.sentinelConfig
              ? (JSON.parse(Buffer.from(runtimeSettings.sentinelConfig).toString()) as Config)
              : undefined,
          }
        : null,
    };
  });
