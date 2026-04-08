import { keyDetailsQueryTimeseriesPayload } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { transformVerificationFilters } from "../../api/keys/timeseries.utils";

export const keyDetailsVerificationsTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keyDetailsQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    // Verify the key belongs to this workspace
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.id, input.keyId),
            eq(table.keyAuthId, input.keyspaceId),
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        columns: { id: true },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve key details due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
        });
      });

    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Key not found in the specified workspace.",
      });
    }

    const { params: transformedInputs, granularity } = transformVerificationFilters({
      ...input,
      names: null,
      keyIds: null,
      identities: null,
      apiId: "",
    });

    const result = await clickhouse.verifications.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: input.keyspaceId,
      keyId: input.keyId,
    });

    return {
      timeseries: result,
      granularity,
    };
  });
