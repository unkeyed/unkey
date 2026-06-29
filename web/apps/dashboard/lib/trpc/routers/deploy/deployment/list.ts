import { and, db, desc, eq, gte, inArray, lte, sql } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import type { LastExit } from "@/lib/types/deploy";
import { TRPCError } from "@trpc/server";
import {
  appRegionalSettings,
  deploymentSteps,
  deployments,
  instances,
  openapiSpecs,
  regions,
} from "@unkey/db/src/schema";
import { z } from "zod";
import { type FlagCode, mapRegionToFlag } from "../network/utils";
import {
  deploymentSelectFields,
  fetchCurrentDeploymentOutsideWindow,
  mapInstanceRow,
  normalizeDeploymentRow,
} from "./deployment-query-helpers";

export const listDeployments = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
      appId: z.string().optional(),
      startTime: z.number().int().optional(),
      endTime: z.number().int().optional(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const deploymentRows = await db
        .select({
          ...deploymentSelectFields,
          appId: deployments.appId,
        })
        .from(deployments)
        .where(
          and(
            eq(deployments.workspaceId, ctx.workspace.id),
            eq(deployments.projectId, input.projectId),
            input.appId ? eq(deployments.appId, input.appId) : undefined,
            input.startTime ? gte(deployments.createdAt, input.startTime) : undefined,
            input.endTime ? lte(deployments.createdAt, input.endTime) : undefined,
          ),
        )
        .orderBy(desc(deployments.createdAt), desc(deployments.id))
        .limit(100);

      if (deploymentRows.length === 0) {
        return [];
      }

      if (
        input.appId !== undefined &&
        input.startTime === undefined &&
        input.endTime === undefined
      ) {
        const currentDeployment = await fetchCurrentDeploymentOutsideWindow(
          ctx.workspace.id,
          { projectId: input.projectId, appId: input.appId },
          deploymentRows,
        );
        if (currentDeployment) {
          deploymentRows.push(currentDeployment);
        }
      }

      const deploymentIds = deploymentRows.map((d) => d.id);

      const appIds = [...new Set(deploymentRows.map((d) => d.appId))];
      const environmentIds = [...new Set(deploymentRows.map((d) => d.environmentId))];

      const [specRows, instanceRows, regionalSettingsRows, stepTimingRows] = await Promise.all([
        db
          .select({ deploymentId: openapiSpecs.deploymentId })
          .from(openapiSpecs)
          .where(inArray(openapiSpecs.deploymentId, deploymentIds)),
        db
          .select({
            id: instances.id,
            deploymentId: instances.deploymentId,
            regionId: regions.id,
            regionName: regions.name,
            regionPlatform: regions.platform,
            status: instances.status,
            containerStatus: instances.containerStatus,
          })
          .from(instances)
          .innerJoin(regions, eq(regions.id, instances.regionId))
          .where(inArray(instances.deploymentId, deploymentIds)),
        db
          .select({
            appId: appRegionalSettings.appId,
            environmentId: appRegionalSettings.environmentId,
            regionId: regions.id,
            regionName: regions.name,
            regionPlatform: regions.platform,
            replicas: appRegionalSettings.replicas,
          })
          .from(appRegionalSettings)
          .innerJoin(regions, eq(regions.id, appRegionalSettings.regionId))
          .where(
            and(
              eq(appRegionalSettings.workspaceId, ctx.workspace.id),
              inArray(appRegionalSettings.appId, appIds),
              inArray(appRegionalSettings.environmentId, environmentIds),
            ),
          ),
        // Build/deploy timing comes from deployment_steps, the only timestamps
        // stop/wake never mutate. openSteps counts steps still running so we
        // can tell an in-progress build (tick live) from a finished one.
        db
          .select({
            deploymentId: deploymentSteps.deploymentId,
            maxEndedAt: sql<number | null>`max(${deploymentSteps.endedAt})`,
            openSteps: sql<number>`sum(case when ${deploymentSteps.endedAt} is null then 1 else 0 end)`,
          })
          .from(deploymentSteps)
          .where(inArray(deploymentSteps.deploymentId, deploymentIds))
          .groupBy(deploymentSteps.deploymentId),
      ]);

      // buildEndedAt is the moment the pipeline finished: the latest step end,
      // but null while any step is still open so the row ticks live instead of
      // freezing a partial duration. Null when a deployment has no steps (old
      // or prebuilt-image rows) — the row then shows no duration.
      const buildEndedAtByDeployment = new Map<string, number | null>();
      for (const row of stepTimingRows) {
        const openSteps = Number(row.openSteps ?? 0);
        const maxEndedAt = row.maxEndedAt == null ? null : Number(row.maxEndedAt);
        buildEndedAtByDeployment.set(row.deploymentId, openSteps > 0 ? null : maxEndedAt);
      }

      const specSet = new Set(specRows.map((s) => s.deploymentId));
      const instancesByDeployment = new Map<string, ReturnType<typeof mapInstanceRow>[]>();
      // The header badge picks the most recent exit across all instances of
      // a deployment so that a multi-region rollout with one OOM-ing pod
      // still surfaces the failure even when others are healthy. Tie-break
      // by finishedAt (preferred) and fall back to the live waiting reason
      // (CrashLoopBackOff, ImagePullBackOff, …) when there's no exit yet.
      const lastExitByDeployment = new Map<string, LastExit>();
      for (const row of instanceRows) {
        const entry = mapInstanceRow(row);
        const list = instancesByDeployment.get(row.deploymentId);
        if (list) {
          list.push(entry);
        } else {
          instancesByDeployment.set(row.deploymentId, [entry]);
        }

        const status = row.containerStatus ?? {};
        const term = status.lastTerminationState ?? null;
        const waiting = status.waiting ?? null;
        const candidate: LastExit = {
          restartCount: status.restartCount ?? 0,
          exitCode: term?.exitCode ?? null,
          signal: term?.signal ?? null,
          reason: term?.reason ?? null,
          finishedAt: term?.finishedAt ?? null,
          statusReason: waiting?.reason ?? null,
        };
        if (candidate.reason === null && candidate.statusReason === null) {
          continue;
        }
        const prev = lastExitByDeployment.get(row.deploymentId);
        if (!prev) {
          lastExitByDeployment.set(row.deploymentId, candidate);
          continue;
        }
        // Prefer the candidate with a more recent finishedAt; if neither
        // has one (only statusReason populated) keep whichever has higher
        // restartCount as a coarse recency tiebreaker.
        const prevTs = prev.finishedAt ?? -1;
        const candTs = candidate.finishedAt ?? -1;
        if (candTs > prevTs || (candTs === prevTs && candidate.restartCount > prev.restartCount)) {
          lastExitByDeployment.set(row.deploymentId, candidate);
        }
      }

      const desiredStateByAppEnv = new Map<
        string,
        {
          desiredInstanceCount: number;
          desiredRegions: {
            region: { id: string; name: string; platform: string };
            flagCode: FlagCode;
          }[];
        }
      >();
      for (const row of regionalSettingsRows) {
        const key = `${row.appId}:${row.environmentId}`;
        const regionEntry = {
          region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
          flagCode: mapRegionToFlag(row.regionName),
        };
        const replicaCount = row.replicas;
        const existing = desiredStateByAppEnv.get(key);
        if (existing) {
          existing.desiredInstanceCount += replicaCount;
          existing.desiredRegions.push(regionEntry);
        } else {
          desiredStateByAppEnv.set(key, {
            desiredInstanceCount: replicaCount,
            desiredRegions: [regionEntry],
          });
        }
      }

      return deploymentRows.map(({ appId, ...deployment }) => {
        const desired = desiredStateByAppEnv.get(`${appId}:${deployment.environmentId}`);
        return {
          ...deployment,
          appId,
          ...normalizeDeploymentRow(deployment),
          instances: instancesByDeployment.get(deployment.id) ?? [],
          buildEndedAt: buildEndedAtByDeployment.get(deployment.id) ?? null,
          lastExit: lastExitByDeployment.get(deployment.id) ?? null,
          desiredInstanceCount: desired?.desiredInstanceCount ?? 0,
          desiredRegions: desired?.desiredRegions ?? [],
          hasOpenApiSpec: specSet.has(deployment.id),
        };
      });
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
