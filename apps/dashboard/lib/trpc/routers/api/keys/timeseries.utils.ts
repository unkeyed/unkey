import type { KeysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { VerificationTimeseriesParams } from "@unkey/clickhouse/src/verifications";
import {
  type TimeseriesConfig,
  type VerificationTimeseriesGranularity,
  getTimeseriesGranularity,
} from "../../utils/granularity";
export function transformVerificationFilters(params: KeysOverviewQueryTimeseriesPayload): {
  params: Omit<VerificationTimeseriesParams, "workspaceId" | "keyspaceId" | "keyId">;
  granularity: VerificationTimeseriesGranularity;
} {
  let timeConfig: TimeseriesConfig<"forVerifications">;

  if (params.since !== "") {
    const startTime = getTimestampFromRelative(params.since);
    const endTime = Date.now();

    timeConfig = getTimeseriesGranularity("forVerifications", startTime, endTime);
  } else {
    timeConfig = getTimeseriesGranularity("forVerifications", params.startTime, params.endTime);
  }

  return {
    params: {
      startTime: timeConfig.startTime,
      endTime: timeConfig.endTime,
      keyIds:
        params.keyIds?.filters.map((f) => ({
          operator: f.operator as "is" | "contains",
          value: f.value,
        })) || null,
      names:
        params.names?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || null,
      identities:
        params.identities?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || null,
      outcomes:
        params.outcomes?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || null,
    },
    granularity: timeConfig.granularity,
  };
}
