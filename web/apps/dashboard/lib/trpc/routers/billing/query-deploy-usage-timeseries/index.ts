import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { deployUsageTimeseries, deployUsageTimeseriesGroup } from "@unkey/clickhouse";
import { z } from "zod";
import { getDeployUsageQueryPeriod } from "./period";

const usageScope = z.object({
  projectId: z.string(),
  appIds: z.array(z.string()).max(100),
  environmentIds: z.array(z.string()).max(100),
});

const usageQuery = z.object({
  groupBy: deployUsageTimeseriesGroup,
  scope: usageScope,
  monthsAgo: z.union([z.literal(0), z.literal(1), z.literal(2)]),
});

export const queryDeployUsageTimeseries = workspaceProcedure
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
  .output(z.array(deployUsageTimeseries))
  .query(async ({ ctx, input }) => {
    const now = new Date();
    const period = getDeployUsageQueryPeriod({
      now,
      monthsAgo: input.monthsAgo,
      dayStart: input.interval === "hour" ? input.day : undefined,
    });
    if (!period) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "The selected day is outside the billing period.",
      });
    }

    try {
      return await clickhouse.billing.deployUsageTimeseries({
        workspaceId: ctx.workspace.id,
        periodStart: period.start,
        end: period.end,
        interval: input.interval,
        groupBy: input.groupBy,
        projectId: input.scope.projectId,
        appIds: input.scope.appIds,
        environmentIds: input.scope.environmentIds,
      });
    } catch (err) {
      console.error("Failed to query deploy usage timeseries", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch Deploy usage over time. Try again later.",
      });
    }
  });
