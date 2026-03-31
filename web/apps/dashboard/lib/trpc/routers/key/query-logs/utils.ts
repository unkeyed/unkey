import type { KeyDetailsLogsPayload } from "@/components/key-details-logs-table/schema/query-logs.schema";
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

  const tags =
    params.tags?.map((t) => ({
      operator: t.operator,
      value: t.value,
    })) ?? null;

  const page = params.page ?? 1;
  const offset = (page - 1) * params.limit;

  return {
    workspaceId,
    keyId: params.keyId,
    keyspaceId: params.keyspaceId,
    limit: params.limit,
    startTime,
    endTime,
    cursorTime: null,
    offset,
    outcomes,
    tags,
  };
}
