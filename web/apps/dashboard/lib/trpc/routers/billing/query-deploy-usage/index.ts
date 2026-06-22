import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const queryDeployUsageResponse = z.object({
  cpuSeconds: z.number(),
  memoryGiBHours: z.number(),
  diskGiBHours: z.number(),
  egressGiB: z.number(),
});

export type DeployUsageResponse = z.infer<typeof queryDeployUsageResponse>;

/**
 * Month-to-date billable Deploy usage for the workspace, read from the same
 * ClickHouse checkpoint aggregation the hourly billing push uses, so the
 * dashboard shows the quantities that are actually billed.
 */
export const queryDeployUsage = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(queryDeployUsageResponse)
  .query(async ({ ctx }) => {
    const now = new Date();
    const monthStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);

    try {
      return await clickhouse.billing.deployMeterUsage({
        workspaceId: ctx.workspace.id,
        start: monthStart,
        end: now.getTime(),
      });
    } catch (err) {
      console.error("Failed to query deploy usage", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch Deploy usage data. Please try again later.",
      });
    }
  });
