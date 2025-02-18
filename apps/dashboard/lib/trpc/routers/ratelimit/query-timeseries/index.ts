import { ratelimitQueryTimeseriesPayload } from "@/app/(app)/ratelimits/[namespaceId]/logs/components/charts/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { transformRatelimitFilters } from "./utils";

export const queryRatelimitTimeseries = rateLimitedProcedure(ratelimit.update)
  .input(ratelimitQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    // Validate workspace exists and belongs to tenant
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          ratelimitNamespaces: {
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),
            columns: {
              id: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve ratelimit timeseries analytics due to a workspace error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    if (workspace.ratelimitNamespaces.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Namespace not found",
      });
    }

    // Transform input filters and determine granularity
    const { params: transformedInputs, granularity } = transformRatelimitFilters(input);

    // Query clickhouse using our new ratelimit timeseries functions
    const result = await clickhouse.ratelimits.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: workspace.id,
      namespaceId: workspace.ratelimitNamespaces[0].id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve ratelimit timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }

    return { timeseries: result.val, granularity };
  });
