import { and, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { environments, regions, sentinelSubscriptions, sentinels } from "@unkey/db/src/schema";
import { z } from "zod";

// listSentinels returns every sentinel in a project joined with its current
// subscription + tier labels and its region. One row per sentinel (sentinels
// already have a UNIQUE(environment_id, region_id) constraint, so "one row per
// region per environment"). The UI groups by environment in-memory.
export const listSentinels = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    const raw = await db
      .select({
        sentinelId: sentinels.id,
        environmentId: sentinels.environmentId,
        environmentSlug: environments.slug,
        regionId: sentinels.regionId,
        regionName: regions.name,
        desiredReplicas: sentinels.desiredReplicas,
        availableReplicas: sentinels.availableReplicas,
        health: sentinels.health,
        desiredState: sentinels.desiredState,
        deployStatus: sentinels.deployStatus,
        subscriptionId: sentinels.subscriptionId,
        tierId: sentinelSubscriptions.tierId,
        tierVersion: sentinelSubscriptions.tierVersion,
        cpuMillicores: sentinelSubscriptions.cpuMillicores,
        memoryMib: sentinelSubscriptions.memoryMib,
      })
      .from(sentinels)
      .innerJoin(sentinelSubscriptions, eq(sentinelSubscriptions.id, sentinels.subscriptionId))
      .innerJoin(environments, eq(environments.id, sentinels.environmentId))
      .innerJoin(regions, eq(regions.id, sentinels.regionId))
      .where(
        and(eq(sentinels.workspaceId, ctx.workspace.id), eq(sentinels.projectId, input.projectId)),
      );

    // Keep the min-replicas rule mirrored on the server (enforced by the
    // ctrl ChangeReplicas handler) and the client (input `min` attribute).
    // Production sentinels must run 3+ for HA; all other environments
    // default to 1. Shape of the rule is owned by sentinelpolicy.go in
    // the ctrl service.
    return raw.map((r) => ({
      ...r,
      minReplicas: r.environmentSlug === "production" ? 3 : 1,
    }));
  });
