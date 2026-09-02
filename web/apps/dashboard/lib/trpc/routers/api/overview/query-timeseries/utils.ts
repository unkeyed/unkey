import type { VerificationTimeseriesParams } from "@unkey/clickhouse/src/verifications";
import type { VerificationQueryTimeseriesPayload } from "@/app/(app)/[workspaceSlug]/apis/_components/hooks/query-timeseries.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import {
  getTimeseriesGranularity,
  type TimeseriesConfig,
  type TimeseriesGranularity,
} from "../../../utils/granularity";

export function transformVerificationFilters(params: VerificationQueryTimeseriesPayload): {
  params: Omit<VerificationTimeseriesParams, "workspaceId" | "keyspaceId" | "keyId">;
  granularity: TimeseriesGranularity;
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
      identities: [],
      startTime: timeConfig.startTime,
      keyIds: [],
      names: [],
      tags: null,
      outcomes: [],
      endTime: timeConfig.endTime,
    },
    granularity: timeConfig.granularity,
  };
}
