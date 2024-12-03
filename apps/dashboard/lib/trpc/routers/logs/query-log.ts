import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { getLogsClickhousePayload } from "@unkey/clickhouse/src/logs";

export const queryLogs = rateLimitedProcedure(ratelimit.update)
  .input(getLogsClickhousePayload)
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          //TODO: change error message later
          message:
            "We are unable opt you in to this beta feature. Please contact support using support@unkey.dev",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Workspace not found, please contact support using support@unkey.dev.",
      });
    }
    const result = await clickhouse.api.logs(input);
    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    return result.val;
  });
