import { and, db, eq } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

type RuntimeSettings = typeof environmentRuntimeSettings.$inferInsert;

export const updateEnvironmentRuntimeSettings = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      port: z.number().min(2000).max(54000).optional(),
      command: z.array(z.string()).optional(),
      healthcheck: z
        .object({
          method: z.enum(["GET", "POST"]),
          path: z.string(),
          intervalSeconds: z.number().default(10),
          timeoutSeconds: z.number().default(5),
          failureThreshold: z.number().default(3),
          initialDelaySeconds: z.number().default(0),
        })
        .nullable()
        .optional(),
      cpuMillicores: z.number().optional(),
      memoryMib: z.number().optional(),
      replicasPerRegion: z.number().min(1).max(10).optional(),
      regions: z.array(z.string()).optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const regionConfig: Record<string, number> = {};

    if (input.regions !== undefined) {
      const existing = await db.query.environmentRuntimeSettings.findFirst({
        where: and(
          eq(environmentRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(environmentRuntimeSettings.environmentId, input.environmentId),
        ),
      });
      const currentConfig = (existing?.regionConfig as Record<string, number>) ?? {};
      for (const region of input.regions) {
        regionConfig[region] = currentConfig[region] ?? 1;
      }
    } else if (input.replicasPerRegion !== undefined) {
      const regionsEnv = process.env.AVAILABLE_REGIONS ?? "";
      for (const region of regionsEnv.split(",")) {
        regionConfig[region] = input.replicasPerRegion;
      }
    }

    const values: RuntimeSettings = {
      workspaceId: ctx.workspace.id,
      environmentId: input.environmentId,
      port: input.port ?? 8080,
      command: input.command ?? [],
      healthcheck: input.healthcheck ?? undefined,
      cpuMillicores: input.cpuMillicores ?? 256,
      memoryMib: input.memoryMib ?? 256,
      regionConfig: regionConfig ?? {},
      createdAt: Date.now(),
    };

    await db
      .insert(environmentRuntimeSettings)
      .values(values)
      .onDuplicateKeyUpdate({ set: values });
  });
