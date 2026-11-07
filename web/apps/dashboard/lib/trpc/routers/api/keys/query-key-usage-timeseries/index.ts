import { keysListQueryTimeseriesPayload } from "@/components/api-keys-table/components/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import { assertValidTimeRange } from "../../../utils/time-range";

export const keyUsageTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keysListQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    assertValidTimeRange(input.startTime, input.endTime);

    if (input.keyIds.length === 0) {
      return { series: {} as Record<string, VerificationTimeseriesDataPoint[]> };
    }

    try {
      const rows = await clickhouse.verifications.timeseriesByKey.perHour({
        workspaceId: ctx.workspace.id,
        keyspaceId: input.keyAuthId,
        keyIds: input.keyIds,
        startTime: input.startTime,
        endTime: input.endTime,
      });

      const series: Record<string, VerificationTimeseriesDataPoint[]> = {};
      for (const row of rows ?? []) {
        const { keyId, ...point } = row;
        const points = series[keyId] ?? [];
        points.push(point);
        series[keyId] = points;
      }

      return { series };
    } catch (error) {
      console.error("Error fetching timeseries data:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }
  });
