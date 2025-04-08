import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { queryUsageResponse } from "./schemas";

export const queryUsage = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .output(queryUsageResponse)
  .query(async ({ ctx }) => {
    const dateNow = new Date();
    const year = dateNow.getUTCFullYear();
    const month = dateNow.getUTCMonth() + 1;

    const [billableRatelimits, billableVerifications] = await Promise.all([
      clickhouse.billing.billableRatelimits({
        workspaceId: ctx.workspace.id,
        year,
        month,
      }),
      clickhouse.billing.billableVerifications({
        workspaceId: ctx.workspace.id,
        year,
        month,
      }),
    ]);

    if (billableRatelimits === null || billableVerifications === null) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch billing usage data. Please try again later.",
      });
    }

    return {
      billableRatelimits,
      billableVerifications,
      billableTotal: billableRatelimits + billableVerifications,
    };
  });
