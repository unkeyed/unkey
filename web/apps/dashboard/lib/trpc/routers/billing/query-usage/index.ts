import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { queryUsageResponse } from "./schemas";

export const queryUsage = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ period: z.enum(["current", "previous"]) }).optional())
  .output(queryUsageResponse)
  .query(async ({ ctx, input }) => {
    const date = new Date();
    date.setUTCDate(1);
    if (input?.period === "previous") {
      date.setUTCMonth(date.getUTCMonth() - 1);
    }
    const year = date.getUTCFullYear();
    const month = date.getUTCMonth() + 1;

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
