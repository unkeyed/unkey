import { and, db, eq, inArray, notInArray } from "@/lib/db";
import { appRegionalSettings, clusterRegions, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../../trpc";
import { resolveProjectEnvironmentIds } from "../utils";

export const updateRegions = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      regions: z.array(z.string()).min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const envIds = await resolveProjectEnvironmentIds(ctx.workspace.id, input.environmentId);

    // Resolve region names to IDs
    const regions = await db.query.clusterRegions.findMany({
      where: inArray(clusterRegions.name, input.regions),
      columns: { id: true, name: true },
    });
    const regionIds = regions.map((r) => r.id);

    if (regionIds.length !== input.regions.length) {
      const found = new Set(regions.map((r) => r.name));
      const missing = input.regions.filter((r) => !found.has(r));
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Unknown regions: ${missing.join(", ")}`,
      });
    }

    // Get appId from the environment
    const env = await db.query.environments.findFirst({
      where: and(
        eq(environments.id, input.environmentId),
        eq(environments.workspaceId, ctx.workspace.id),
      ),
      columns: { appId: true },
    });
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    // Get existing scaling settings to preserve values for new regions
    const existingSettings = await db.query.appRegionalSettings.findMany({
      where: and(
        eq(appRegionalSettings.workspaceId, ctx.workspace.id),
        eq(appRegionalSettings.environmentId, input.environmentId),
      ),
    });

    const defaults = existingSettings.at(0) ?? {
      replicas: 1,
    };

    for (const envId of envIds) {
      // Delete rows for regions that are no longer selected
      await db.delete(appRegionalSettings).where(
        and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          eq(appRegionalSettings.environmentId, envId),
          notInArray(appRegionalSettings.regionId, regionIds),
        ),
      );

      // Insert rows for newly added regions
      for (const regionId of regionIds) {
        const existing = existingSettings.find((s) => s.regionId === regionId);
        if (!existing) {
          await db.insert(appRegionalSettings).values({
            workspaceId: ctx.workspace.id,
            appId: env.appId,
            environmentId: envId,
            regionId,
            replicas: defaults.replicas,
          });
        }
      }
    }
  });
