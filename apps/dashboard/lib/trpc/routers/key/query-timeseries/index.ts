import { keyDetailsQueryTimeseriesPayload } from "@/app/(app)/[workspace]/apis/[apiId]/keys/[keyAuthId]/[keyId]/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db, isNull } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { transformVerificationFilters } from "../../api/keys/timeseries.utils";

export const keyDetailsVerificationsTimeseries = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(keyDetailsQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          keys: {
            where: (keysTable, { eq, and }) =>
              and(
                eq(keysTable.id, input.keyId),
                eq(keysTable.keyAuthId, input.keyspaceId),
                isNull(keysTable.deletedAtM),
              ),
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
            "Failed to retrieve key details due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
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
