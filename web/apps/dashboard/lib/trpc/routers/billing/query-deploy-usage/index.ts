import { projectDeployUsage, sumDeployMeterCents } from "@/lib/billing/deployPricing";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { unstable_cache } from "next/cache";
import { z } from "zod";

/**
 * Window whose average run-rate we extrapolate to the period end. Recent enough
 * that it reflects what is currently deployed, long enough to smooth spikes.
 */
const TRAILING_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;

const CACHE_TTL_SECONDS = 5 * 60;

export const queryDeployUsageResponse = z.object({
  cpuSeconds: z.number(),
  memoryGiBHours: z.number(),
  diskGiBHours: z.number(),
  egressGiB: z.number(),
  activeKeys: z.number(),
  /**
   * Month-to-date gross usage priced locally (cents), rounded per meter the way
   * the invoice is, so this matches the usage lines Stripe bills.
   */
  grossCents: z.number(),
  /**
   * Gross usage projected to the end of the calendar month (cents): month-to-date
   * plus the trailing-window run-rate over the remaining days. A forecast for
   * display, not a billed figure.
   */
  projectedGrossCents: z.number(),
});

export type DeployUsageResponse = z.infer<typeof queryDeployUsageResponse>;

const fetchUsage = unstable_cache(
  async (workspaceId: string): Promise<DeployUsageResponse> => {
    const now = new Date();
    const nowMs = now.getTime();
    const monthStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
    const monthEnd = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1);

    const [meters, keys] = await Promise.all([
      clickhouse.billing.deployMeterUsage({
        workspaceId,
        periodStart: monthStart,
        trailingStart: nowMs - TRAILING_WINDOW_MS,
        end: nowMs,
      }),
      clickhouse.billing.activeKeysUsage({
        workspaceId,
        year: now.getUTCFullYear(),
        month: now.getUTCMonth() + 1,
      }),
    ]);

    const monthToDate = {
      cpuSeconds: meters.cpuSeconds,
      memoryGiBHours: meters.memoryGiBHours,
      diskGiBHours: meters.diskGiBHours,
      egressGiB: meters.egressGiB,
      activeKeys: keys.activeKeys,
    };
    const trailing = {
      cpuSeconds: meters.trailingCpuSeconds,
      memoryGiBHours: meters.trailingMemoryGiBHours,
      diskGiBHours: meters.trailingDiskGiBHours,
      egressGiB: meters.trailingEgressGiB,
    };

    const projected = projectDeployUsage(
      monthToDate,
      trailing,
      TRAILING_WINDOW_MS,
      monthEnd - nowMs,
    );

    return {
      cpuSeconds: meters.cpuSeconds,
      memoryGiBHours: meters.memoryGiBHours,
      diskGiBHours: meters.diskGiBHours,
      egressGiB: meters.egressGiB,
      activeKeys: keys.activeKeys,
      grossCents: sumDeployMeterCents(monthToDate),
      projectedGrossCents: sumDeployMeterCents(projected),
    };
  },
  ["billing", "deploy-usage"],
  { revalidate: CACHE_TTL_SECONDS },
);

/**
 * Month-to-date billable Deploy usage for the workspace, read from the same
 * ClickHouse checkpoint aggregation the hourly billing push uses, so the
 * dashboard shows the quantities that are actually billed.
 */
export const queryDeployUsage = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(queryDeployUsageResponse)
  .query(async ({ ctx }) => {
    try {
      return await fetchUsage(ctx.workspace.id);
    } catch (err) {
      console.error("Failed to query deploy usage", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch Deploy usage data. Please try again later.",
      });
    }
  });
