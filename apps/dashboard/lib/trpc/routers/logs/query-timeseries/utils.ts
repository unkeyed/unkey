import type { QueryLogsPayload } from "@/app/(app)/logs/filters.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { LogsTimeseriesParams } from "@unkey/clickhouse/src/logs";
import {
  type RegularTimeseriesGranularity,
  type TimeseriesConfig,
  getTimeseriesGranularity,
} from "../../utils/granularity";

export function transformFilters(params: Omit<QueryLogsPayload, "limit" | "cursor">): {
  params: Omit<LogsTimeseriesParams, "workspaceId">;
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
      hosts: params.host?.filters.map((f) => f.value) || [],
      methods: params.methods?.filters.map((f) => f.value) || [],
      paths:
        params.paths?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || [],
      statusCodes: params.status?.filters.map((f) => f.value) || [],
    },
    granularity: timeConfig.granularity,
  };
}
