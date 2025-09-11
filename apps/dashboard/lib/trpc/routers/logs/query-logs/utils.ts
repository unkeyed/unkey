import type { queryLogsPayload } from "@/app/(app)/[workspace]/logs/components/table/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { GetLogsClickhousePayload } from "@unkey/clickhouse/src/logs";
import type { z } from "zod";

export function transformFilters(
  params: z.infer<typeof queryLogsPayload>,
): Omit<GetLogsClickhousePayload, "workspaceId"> {
  // Transform path filters to include operators
  const paths =
    params.path?.filters.map((f) => ({
      operator: f.operator,
      value: f.value,
    })) || [];

  // Extract other filters as before
  const requestIds = params.requestId?.filters.map((f) => f.value) || [];
  const hosts = params.host?.filters.map((f) => f.value) || [];
  const methods = params.method?.filters.map((f) => f.value) || [];
  const statusCodes = params.status?.filters.map((f) => f.value) || [];

  let startTime = params.startTime;
  let endTime = params.endTime;

  // If we have relativeTime filter `since`, ignore other time params
  const hasRelativeTime = params.since !== "";
  if (hasRelativeTime) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  return {
    limit: params.limit,
    startTime,
    endTime,
    requestIds,
    hosts,
    methods,
    paths,
    statusCodes,
    cursorTime: params.cursor ?? null,
  };
}
