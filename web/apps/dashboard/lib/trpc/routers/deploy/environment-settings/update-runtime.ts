import { db } from "@/lib/db";
import { environmentRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

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
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const set: Record<string, unknown> = { updatedAt: Date.now() };
    if (input.port !== undefined) set.port = input.port;
    if (input.command !== undefined) set.command = input.command;
    if (input.healthcheck !== undefined) set.healthcheck = input.healthcheck;
    if (input.cpuMillicores !== undefined) set.cpuMillicores = input.cpuMillicores;
    if (input.memoryMib !== undefined) set.memoryMib = input.memoryMib;
    if (input.replicasPerRegion !== undefined) {
      const regionsEnv = process.env.AVAILABLE_REGIONS ?? "";
      const regions = regionsEnv
        .split(",")
        .map((r) => r.trim())
        .filter(Boolean);
      set.regionConfig = Object.fromEntries(regions.map((r) => [r, input.replicasPerRegion]));
    }

    await db
      .insert(environmentRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        environmentId: input.environmentId,
        port: input.port ?? 8080,
        command: input.command ?? [],
        healthcheck: input.healthcheck ?? undefined,
        cpuMillicores: input.cpuMillicores ?? 256,
        memoryMib: input.memoryMib ?? 256,
        regionConfig: {},
        createdAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set });
  });
