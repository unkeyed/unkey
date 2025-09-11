import type { RatelimitQueryTimeseriesPayload } from "@/app/(app)/[workspace]/ratelimits/[namespaceId]/logs/components/charts/query-timeseries.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RatelimitLogsTimeseriesParams } from "@unkey/clickhouse/src/ratelimits";
import {
  type RegularTimeseriesGranularity,
  type TimeseriesConfig,
  getTimeseriesGranularity,
} from "../../utils/granularity";

export function transformRatelimitFilters(params: RatelimitQueryTimeseriesPayload): {
  params: Omit<RatelimitLogsTimeseriesParams, "workspaceId" | "namespaceId">;
  granularity: RegularTimeseriesGranularity;
} {
  let timeConfig: TimeseriesConfig<"forRegular">;

  if (params.since !== "") {
    const startTime = getTimestampFromRelative(params.since);
    const endTime = Date.now();
    timeConfig = getTimeseriesGranularity("forRegular", startTime, endTime);
  } else {
    timeConfig = getTimeseriesGranularity("forRegular", params.startTime, params.endTime);
  }

  return {
    params: {
      startTime: timeConfig.startTime,
      endTime: timeConfig.endTime,
      identifiers:
        params.identifiers?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || null,
    },
    granularity: timeConfig.granularity,
  };
}
