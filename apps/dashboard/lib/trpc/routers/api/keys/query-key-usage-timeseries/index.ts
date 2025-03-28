import { keysListQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/keys_v2/[keyAuthId]/_components/components/table/components/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";

export const keyUsageTimeseries = rateLimitedProcedure(ratelimit.read)
  .input(keysListQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const result = await clickhouse.verifications.timeseries.perHour({
      workspaceId: ctx.workspace.id,
      endTime: input.endTime,
      startTime: input.startTime,
      keyId: input.keyId,
      keyspaceId: input.keyAuthId,
      identities: null,
      keyIds: null,
      names: null,
      outcomes: null,
    });

    return {
      timeseries: result,
    };
  });
