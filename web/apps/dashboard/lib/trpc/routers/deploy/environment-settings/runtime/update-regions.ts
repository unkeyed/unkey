import { and, db, eq, inArray, notInArray } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRegionalSettings, environments, regions } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

// Per-id length matches the regions.id varchar(64); the array cap bounds the
// delete/insert fanout per request. No deployment targets anywhere near 50
// regions, so this is an abuse bound rather than a functional limit.
const MAX_REGIONS_PER_REQUEST = 50;
const MAX_REGION_ID_LENGTH = 64;

export const updateRegions = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      regionIds: z
        .array(z.string().min(1).max(MAX_REGION_ID_LENGTH))
        .min(1)
        .max(MAX_REGIONS_PER_REQUEST),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // The client can submit the same region twice; dedup before persisting so
    // the insert below cannot violate the unique_app_env_region index.
    const requestedRegionIds = [...new Set(input.regionIds)];

    // None of these reads depend on each other, so issue them together.
    const [env, knownRegions, existingSettings] = await Promise.all([
      db.query.environments.findFirst({
        where: and(
          eq(environments.id, input.environmentId),
          eq(environments.workspaceId, ctx.workspace.id),
        ),
        columns: { appId: true },
      }),
      db.query.regions.findMany({
        where: inArray(regions.id, requestedRegionIds),
        columns: { id: true, canSchedule: true },
      }),
      db.query.appRegionalSettings.findMany({
        where: and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          eq(appRegionalSettings.environmentId, input.environmentId),
        ),
      }),
    ]);
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    const existingRegionIds = new Set(existingSettings.map((s) => s.regionId));

    // A region already assigned to this environment is always allowed to remain,
    // even if it has since gone unschedulable or been removed from the regions
    // table entirely. There is no FK from app_regional_settings.region_id, so a
    // decommissioned region can linger here; the UI keeps such regions selected
    // (with a warning), and rejecting them would lock the user out of region
    // management. Newly added regions must both exist and be schedulable, which
    // membership in schedulableRegionIds (known AND canSchedule) already covers.
    const schedulableRegionIds = new Set(
      knownRegions.filter((region) => region.canSchedule).map((region) => region.id),
    );
    const invalidRegionIds = requestedRegionIds.filter((id) => {
      if (existingRegionIds.has(id)) {
        return false;
      }
      return !schedulableRegionIds.has(id);
    });
    if (invalidRegionIds.length > 0) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Unknown or unschedulable region(s): ${invalidRegionIds.join(", ")}`,
      });
    }

    const defaultReplicas = existingSettings.at(0)?.replicas ?? 1;
    const defaultPolicyId = existingSettings.at(0)?.horizontalAutoscalingPolicyId ?? null;

    const toInsert = requestedRegionIds
      .filter((regionId) => !existingRegionIds.has(regionId))
      .map((regionId) => ({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        regionId,
        replicas: defaultReplicas,
        horizontalAutoscalingPolicyId: defaultPolicyId,
      }));

    // The delete and insert must be atomic: a failed insert after a committed
    // delete would leave the environment with regions removed but their
    // replacements never written.
    await db.transaction(async (tx) => {
      await tx
        .delete(appRegionalSettings)
        .where(
          and(
            eq(appRegionalSettings.workspaceId, ctx.workspace.id),
            eq(appRegionalSettings.environmentId, input.environmentId),
            notInArray(appRegionalSettings.regionId, requestedRegionIds),
          ),
        );

      if (toInsert.length > 0) {
        await tx.insert(appRegionalSettings).values(toInsert);
      }
    });
  });
