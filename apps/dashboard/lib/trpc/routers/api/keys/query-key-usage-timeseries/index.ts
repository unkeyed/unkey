import { keysListQueryTimeseriesPayload } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

export const keyUsageTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keysListQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    try {
      const result = await clickhouse.verifications.timeseries.perHour({
        workspaceId: ctx.workspace.id,
        endTime: input.endTime,
        startTime: input.startTime,
        keyspaceId: input.keyAuthId,
        keyId: input.keyId,
        identities: null,
        keyIds: null,
        names: null,
        tags: null,
        outcomes: null,
      });

      if (!result || result.length === 0) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Something went wrong when fetching data from ClickHouse.",
        });
      }

      return {
        timeseries: result,
      };
    } catch (error) {
      console.error("Error fetching timeseries data:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }
  });
