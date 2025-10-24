import type { TimeseriesRequestSchema } from "@/lib/schemas/logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { LogsTimeseriesParams } from "@unkey/clickhouse/src/logs";
import {
  type TimeseriesConfig,
  type TimeseriesGranularity,
  getTimeseriesGranularity,
} from "../../utils/granularity";

export function transformFilters(params: TimeseriesRequestSchema): {
  params: Omit<LogsTimeseriesParams, "workspaceId">;
  granularity: TimeseriesGranularity;
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
      hosts: params.host?.filters?.map((f) => f.value) || [],
      excludeHosts: params.host?.exclude || [],
      methods: params.method?.filters.map((f) => f.value) || [],
      paths:
        params.path?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || [],
      statusCodes: params.status?.filters.map((f) => f.value) || [],
    },
    granularity: timeConfig.granularity,
  };
}
