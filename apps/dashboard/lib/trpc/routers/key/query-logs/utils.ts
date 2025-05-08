import type { KeyDetailsLogsPayload } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/[keyId]/components/table/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { KeyDetailsLogsParams } from "@unkey/clickhouse/src/verifications";

export function transformKeyDetailsFilters(
  params: KeyDetailsLogsPayload,
  workspaceId: string,
): KeyDetailsLogsParams {
  let startTime = params.startTime;
  let endTime = params.endTime;

  if (params.since) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  const outcomes =
    params.outcomes?.map((o) => ({
      operator: "is" as const,
      value: o.value,
    })) ?? null;

  return {
    workspaceId,
    keyId: params.keyId,
    keyspaceId: params.keyspaceId,
    limit: params.limit,
    startTime,
    endTime,
    cursorTime: params.cursor ?? null,
    outcomes,
  };
}
