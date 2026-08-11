import { priceComputeMeterMicroCents } from "@/lib/billing/deployPricing";
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

export const queryDeployUsageBreakdownResponse = z.array(deployUsageBreakdownRow);

export type DeployUsageBreakdownRow = z.infer<typeof deployUsageBreakdownRow>;

export const queryDeployUsageBreakdown = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(queryDeployUsageBreakdownResponse)
  .query(async ({ ctx }) => {
    const now = new Date();
    const monthStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
    const monthEnd = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1);

    try {
      const usage = await clickhouse.billing.deployUsageByScope({
        workspaceId: ctx.workspace.id,
        periodStart: monthStart,
        end: Math.min(now.getTime(), monthEnd),
      });

      if (usage.length === 0) {
        return [];
      }

      const ids = (values: Array<string>) => Array.from(new Set(values.filter((v) => v !== "")));
      const projectIds = ids(usage.map((row) => row.projectId));
      const appIds = ids(usage.map((row) => row.appId));
      const environmentIds = ids(usage.map((row) => row.environmentId));

      const [projects, apps, environments] = await Promise.all([
        projectIds.length === 0
          ? []
          : db
              .select({ id: schema.projects.id, name: schema.projects.name })
              .from(schema.projects)
              .where(
                and(
                  eq(schema.projects.workspaceId, ctx.workspace.id),
                  inArray(schema.projects.id, projectIds),
                ),
              ),
        appIds.length === 0
          ? []
          : db
              .select({ id: schema.apps.id, name: schema.apps.name })
              .from(schema.apps)
              .where(
                and(eq(schema.apps.workspaceId, ctx.workspace.id), inArray(schema.apps.id, appIds)),
              ),
        environmentIds.length === 0
          ? []
          : db
              .select({ id: schema.environments.id, slug: schema.environments.slug })
              .from(schema.environments)
              .where(
                and(
                  eq(schema.environments.workspaceId, ctx.workspace.id),
                  inArray(schema.environments.id, environmentIds),
                ),
              ),
      ]);

      const projectNames = new Map(projects.map((p) => [p.id, p.name]));
      const appNames = new Map(apps.map((a) => [a.id, a.name]));
      const environmentSlugs = new Map(environments.map((e) => [e.id, e.slug]));

      return usage.map((row) => ({
        projectId: row.projectId,
        projectName: projectNames.get(row.projectId) ?? null,
        appId: row.appId,
        appName: appNames.get(row.appId) ?? null,
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
      }));
    } catch (err) {
      console.error("Failed to query deploy usage breakdown", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch Deploy usage breakdown. Please try again later.",
      });
    }
  });
