import { DEPLOYMENT_STATUSES } from "@/lib/collections/deploy/deployment-status";
import { and, db, desc, eq, gte, inArray, lt, lte, or, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
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
  computeLastExit,
  deploymentSelectFields,
  mapInstanceRow,
  normalizeDeploymentRow,
} from "./deployment-query-helpers";

const MAX_LIMIT = 500;

export const listDeployments = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
      appId: z.string().optional(),
      deploymentIds: z.array(z.string()).min(1).max(100).optional(),
      environmentIds: z.array(z.string()).min(1).max(50).optional(),
      statuses: z.array(z.enum(DEPLOYMENT_STATUSES)).min(1).optional(),
      branches: z.array(z.string()).min(1).max(50).optional(),
      startTime: z.number().int().optional(),
      endTime: z.number().int().optional(),
      limit: z.number().int().min(1).max(MAX_LIMIT).default(100),
      // The last row of the previous page. A keyset rather than a row offset,
      // so a deployment created between two page loads cannot shift a row out
      // of every page. Named `cursor` because that is the page parameter
      // tRPC's useInfiniteQuery injects.
      cursor: z.object({ createdAt: z.number().int(), id: z.string() }).nullish(),
    }),
  )
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    try {
      // One extra row tells the client whether another page exists without a
      // second count query.
      const rows = await db
        .select({
          ...deploymentSelectFields,
          appId: deployments.appId,
        })
        .from(deployments)
        .where(
          and(
            eq(deployments.workspaceId, ctx.workspace.id),
            eq(deployments.projectId, input.projectId),
            input.appId !== undefined ? eq(deployments.appId, input.appId) : undefined,
            input.deploymentIds ? inArray(deployments.id, input.deploymentIds) : undefined,
            input.environmentIds
              ? inArray(deployments.environmentId, input.environmentIds)
              : undefined,
            input.statuses ? inArray(deployments.status, input.statuses) : undefined,
            input.branches ? inArray(deployments.gitBranch, input.branches) : undefined,
            input.startTime !== undefined ? gte(deployments.createdAt, input.startTime) : undefined,
            input.endTime !== undefined ? lte(deployments.createdAt, input.endTime) : undefined,
            input.cursor
              ? or(
                  lt(deployments.createdAt, input.cursor.createdAt),
                  and(
                    eq(deployments.createdAt, input.cursor.createdAt),
                    lt(deployments.id, input.cursor.id),
                  ),
                )
              : undefined,
          ),
        )
        .orderBy(desc(deployments.createdAt), desc(deployments.id))
        .limit(input.limit + 1);

      const hasMore = rows.length > input.limit;
      const deploymentRows = hasMore ? rows.slice(0, input.limit) : rows;
      const last = deploymentRows.at(-1);
      const nextCursor = hasMore && last ? { createdAt: last.createdAt, id: last.id } : null;

      if (deploymentRows.length === 0) {
        return { deployments: [], nextCursor: null };
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
      // Group raw rows per deployment so the header "OOMKilled · exit=137"
      // badge can be derived with the shared computeLastExit helper, the same
      // logic getById uses for the single-deployment view.
      const rowsByDeployment = new Map<string, typeof instanceRows>();
      for (const row of instanceRows) {
        const entry = mapInstanceRow(row);
        const list = instancesByDeployment.get(row.deploymentId);
        if (list) {
          list.push(entry);
        } else {
          instancesByDeployment.set(row.deploymentId, [entry]);
        }
        const rows = rowsByDeployment.get(row.deploymentId);
        if (rows) {
          rows.push(row);
        } else {
          rowsByDeployment.set(row.deploymentId, [row]);
        }
      }
      const lastExitByDeployment = new Map<string, LastExit>();
      for (const [deploymentId, rows] of rowsByDeployment) {
        const lastExit = computeLastExit(rows);
        if (lastExit) {
          lastExitByDeployment.set(deploymentId, lastExit);
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

      const enriched = deploymentRows.map(({ appId, ...deployment }) => {
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

      return { deployments: enriched, nextCursor };
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
