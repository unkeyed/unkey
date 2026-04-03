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
    const workspace = await db.query.workspaces
      .findFirst({
        where: { orgId: ctx.tenant.id, deletedAtM: { isNull: true } },
        with: {
          keys: {
            where: { id: input.keyId, keyAuthId: input.keyspaceId, deletedAtM: { isNull: true } },
            limit: 1,
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
            "Failed to retrieve key details due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.com.",
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
