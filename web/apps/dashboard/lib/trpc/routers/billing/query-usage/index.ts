import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { queryUsageResponse } from "./schemas";

export const queryUsage = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ monthsAgo: z.union([z.literal(0), z.literal(1)]) }).optional())
  .output(queryUsageResponse)
  .query(async ({ ctx, input }) => {
    const dateNow = new Date();
    const period = new Date(
      Date.UTC(dateNow.getUTCFullYear(), dateNow.getUTCMonth() - (input?.monthsAgo ?? 0), 1),
    );
    const year = period.getUTCFullYear();
    const month = period.getUTCMonth() + 1;

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
      billableRatelimits: billableRatelimits,
      billableVerifications: billableVerifications,
      billableTotal: billableRatelimits + billableVerifications,
    };
  });
