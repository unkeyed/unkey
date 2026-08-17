import {
  priceActiveKeysMicroCents,
  priceComputeMeterMicroCents,
} from "@/lib/billing/deployPricing";
import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deployUsageBreakdownRow = z.object({
  projectId: z.string(),
  projectName: z.string().nullable(),
  appId: z.string(),
  appName: z.string().nullable(),
  environmentId: z.string(),
  environmentSlug: z.string().nullable(),
  cpuSeconds: z.number(),
  memoryGiBHours: z.number(),
  diskGiBHours: z.number(),
  egressGiB: z.number(),
  grossMicroCents: z.number(),
});

export const deployGatewayRow = z.object({
  projectId: z.string(),
  projectName: z.string().nullable(),
  appId: z.string(),
  activeKeys: z.number(),
  grossMicroCents: z.number(),
});

export const queryDeployUsageBreakdownResponse = z.object({
  usage: z.array(deployUsageBreakdownRow),
  gateway: z.array(deployGatewayRow),
});

export type DeployUsageBreakdownRow = z.infer<typeof deployUsageBreakdownRow>;
export type DeployUsageBreakdown = z.infer<typeof queryDeployUsageBreakdownResponse>;

async function resolveScopeNames(
  workspaceId: string,
  usage: Array<{ projectId: string; appId: string; environmentId: string }>,
  keys: Array<{ appId: string }>,
) {
  const ids = (...lists: Array<Array<string>>) =>
    Array.from(new Set(lists.flat().filter((value) => value !== "")));
  const environmentIds = ids(usage.map((row) => row.environmentId));
  const environments =
    environmentIds.length === 0
      ? []
      : await db
          .select({
            id: schema.environments.id,
            slug: schema.environments.slug,
            appId: schema.environments.appId,
          })
          .from(schema.environments)
          .where(
            and(
              eq(schema.environments.workspaceId, workspaceId),
              inArray(schema.environments.id, environmentIds),
            ),
          );

  const environmentApps = new Map(
    environments.map((environment) => [environment.id, environment.appId]),
  );
  // Gateway apps need resolving too: an app can verify keys in a period it ran
  // no compute in, so it appears here without a usage row to carry it.
  const appIds = ids(
    usage.map((row) =>
      row.appId === "" ? (environmentApps.get(row.environmentId) ?? "") : row.appId,
    ),
    keys.map((row) => row.appId),
  );
  const apps =
    appIds.length === 0
      ? []
      : await db
          .select({
            id: schema.apps.id,
            name: schema.apps.name,
            projectId: schema.apps.projectId,
          })
          .from(schema.apps)
          .where(and(eq(schema.apps.workspaceId, workspaceId), inArray(schema.apps.id, appIds)));

  const appProjects = new Map(apps.map((app) => [app.id, app.projectId]));

  const projectIds = ids(
    usage.map((row) => row.projectId),
    keys.map((row) => appProjects.get(row.appId) ?? ""),
  );
  const projects =
    projectIds.length === 0
      ? []
      : await db
          .select({ id: schema.projects.id, name: schema.projects.name })
          .from(schema.projects)
          .where(
            and(
              eq(schema.projects.workspaceId, workspaceId),
              inArray(schema.projects.id, projectIds),
            ),
          );

  return {
    appProjects,
    environmentApps,
    projectNames: new Map(projects.map((project) => [project.id, project.name])),
    appNames: new Map(apps.map((app) => [app.id, app.name])),
    environmentSlugs: new Map(
      environments.map((environment) => [environment.id, environment.slug]),
    ),
  };
}

export const queryDeployUsageBreakdown = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(queryDeployUsageBreakdownResponse)
  .query(async ({ ctx }) => {
    const now = new Date();
    const monthStartMillis = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
    const monthEndMillis = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1);

    try {
      const [usage, keys] = await Promise.all([
        clickhouse.billing.deployUsageByScope({
          workspaceId: ctx.workspace.id,
          periodStart: monthStartMillis,
          end: Math.min(now.getTime(), monthEndMillis),
        }),
        clickhouse.billing.activeKeysByApp({
          workspaceId: ctx.workspace.id,
          year: now.getUTCFullYear(),
          month: now.getUTCMonth() + 1,
        }),
      ]);

      if (usage.length === 0 && keys.length === 0) {
        return { usage: [], gateway: [] };
      }

      const { appProjects, environmentApps, projectNames, appNames, environmentSlugs } =
        await resolveScopeNames(ctx.workspace.id, usage, keys);

      const gateway = keys.map((row) => {
        const projectId = appProjects.get(row.appId) ?? "";
        return {
          projectId,
          projectName: projectNames.get(projectId) ?? null,
          appId: row.appId,
          activeKeys: row.activeKeys,
          grossMicroCents: priceActiveKeysMicroCents(row.activeKeys),
        };
      });

      const usageRows = usage.map((row) => {
        // TODO: Remove this fallback after all instance usage rows include app_id.
        const appId = row.appId === "" ? (environmentApps.get(row.environmentId) ?? "") : row.appId;

        return {
          projectId: row.projectId,
          projectName: projectNames.get(row.projectId) ?? null,
          appId,
          appName: appNames.get(appId) ?? null,
          environmentId: row.environmentId,
          environmentSlug: environmentSlugs.get(row.environmentId) ?? null,
          cpuSeconds: row.cpuSeconds,
          memoryGiBHours: row.memoryGiBHours,
          diskGiBHours: row.diskGiBHours,
          egressGiB: row.egressGiB,
          grossMicroCents: priceComputeMeterMicroCents({
            cpuSeconds: row.cpuSeconds,
            memoryGiBHours: row.memoryGiBHours,
            diskGiBHours: row.diskGiBHours,
            egressGiB: row.egressGiB,
          }),
        };
      });

      return { usage: usageRows, gateway };
    } catch (err) {
      console.error("Failed to query deploy usage breakdown", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch Deploy usage breakdown. Please try again later.",
      });
    }
  });
