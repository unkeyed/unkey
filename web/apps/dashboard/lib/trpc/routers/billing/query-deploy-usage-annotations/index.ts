import { and, db, eq, gte, inArray, lt, or, schema, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getDeployUsageQueryPeriod } from "../query-deploy-usage-timeseries/period";

const DEPLOYMENT_IDS_PER_BUCKET = 5;
const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;

const usageScope = z.object({
  projectId: z.string(),
  appIds: z.array(z.string()).max(100),
  environmentIds: z.array(z.string()).max(100),
});

const usageQuery = z.object({
  scope: usageScope,
  monthsAgo: z.union([z.literal(0), z.literal(1), z.literal(2)]),
});

const deploymentAnnotation = z.object({
  time: z.number(),
  count: z.number().int(),
  deploymentIds: z.array(z.string()),
});

export const queryDeployUsageAnnotations = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.discriminatedUnion("interval", [
      usageQuery.extend({ interval: z.literal("day") }),
      usageQuery.extend({
        interval: z.literal("hour"),
        day: z.number().int().nonnegative(),
      }),
    ]),
  )
  .output(z.array(deploymentAnnotation))
  .query(async ({ ctx, input }) => {
    const period = getDeployUsageQueryPeriod({
      now: new Date(),
      monthsAgo: input.monthsAgo,
      dayStart: input.interval === "hour" ? input.day : undefined,
    });
    if (!period) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "The selected day is outside the billing period.",
      });
    }

    const bucketMs = input.interval === "hour" ? HOUR_MS : DAY_MS;
    const bucket = sql<number>`floor(${schema.deployments.createdAt} / ${bucketMs}) * ${bucketMs}`;
    const selectedScope =
      input.scope.appIds.length === 0 && input.scope.environmentIds.length === 0
        ? undefined
        : or(
            input.scope.appIds.length > 0
              ? inArray(schema.deployments.appId, input.scope.appIds)
              : undefined,
            input.scope.environmentIds.length > 0
              ? inArray(schema.deployments.environmentId, input.scope.environmentIds)
              : undefined,
          );

    try {
      const rows = await db
        .select({
          time: bucket,
          count: sql<number>`count(*)`,
          deploymentIds: sql<string>`substring_index(group_concat(${schema.deployments.id} order by ${schema.deployments.createdAt} desc separator ','), ',', ${DEPLOYMENT_IDS_PER_BUCKET})`,
        })
        .from(schema.deployments)
        .where(
          and(
            eq(schema.deployments.workspaceId, ctx.workspace.id),
            gte(schema.deployments.createdAt, period.start),
            lt(schema.deployments.createdAt, period.end),
            input.scope.projectId
              ? eq(schema.deployments.projectId, input.scope.projectId)
              : undefined,
            selectedScope,
          ),
        )
        .groupBy(bucket)
        .orderBy(bucket);

      return rows.map((row) => ({
        time: Number(row.time),
        count: Number(row.count),
        deploymentIds: row.deploymentIds.split(","),
      }));
    } catch (err) {
      console.error("Failed to query Deploy usage annotations", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment annotations. Try again later.",
      });
    }
  });
