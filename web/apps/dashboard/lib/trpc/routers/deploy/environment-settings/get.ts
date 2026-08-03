import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appBuildSettings, appRegionalSettings, appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

export const getEnvironmentSettings = workspaceProcedure
  .input(z.object({ environmentId: z.string() }))
  .query(async ({ ctx, input }) => {
    try {
      const [buildSettings, runtimeSettings, regionalSettings] = await Promise.all([
        db.query.appBuildSettings.findFirst({
          where: and(
            eq(appBuildSettings.workspaceId, ctx.workspace.id),
            eq(appBuildSettings.environmentId, input.environmentId),
          ),
          columns: {
            autoDeploy: true,
            dockerfile: true,
            dockerContext: true,
            buildCommand: true,
            watchPaths: true,
          },
        }),
        db.query.appRuntimeSettings.findFirst({
          where: and(
            eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
            eq(appRuntimeSettings.environmentId, input.environmentId),
          ),
          columns: {
            port: true,
            cpuMillicores: true,
            memoryMib: true,
            storageMib: true,
            command: true,
            healthcheck: true,
            upstreamProtocol: true,
            openapiSpecPath: true,
          },
        }),
        db.query.appRegionalSettings.findMany({
          where: and(
            eq(appRegionalSettings.workspaceId, ctx.workspace.id),
            eq(appRegionalSettings.environmentId, input.environmentId),
          ),
          columns: {
            replicas: true,
          },
          with: {
            region: {
              columns: {
                id: true,
                name: true,
              },
            },
            horizontalAutoscalingPolicy: {
              columns: {
                replicasMin: true,
              },
            },
          },
        }),
      ]);

      return {
        buildSettings: buildSettings ?? null,
        runtimeSettings: runtimeSettings ?? null,
        regionalSettings,
      };
    } catch (err) {
      console.error(err);
      if (err instanceof TRPCError) {
        throw err;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Unable to load environment.",
      });
    }
  });
