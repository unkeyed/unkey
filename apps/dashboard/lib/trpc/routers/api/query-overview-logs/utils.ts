import { getTimestampFromRelative } from "@/lib/utils";
import type { KeysOverviewLogsParams } from "@unkey/clickhouse/src/keys/keys";
import type { KeysQueryOverviewLogsPayload } from ".";

export function transformKeysFilters(
  params: KeysQueryOverviewLogsPayload
): Omit<KeysOverviewLogsParams, "workspaceId" | "keyspaceId"> {
  let startTime = params.startTime;
  let endTime = params.endTime;

  if (params.since) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  const keyIds =
    params.keyIds?.map((k) => ({
      operator: k.operator,
      value: k.value,
    })) ?? null;

  const outcomes =
    params.outcomes?.map((o) => ({
      operator: "is" as const,
      value: o.value,
    })) ?? null;

  const sorts =
    params.sorts?.map((sort) => ({
      column: sort.column,
      direction: sort.direction,
    })) ?? null;

  return {
    limit: params.limit,
    startTime,
    endTime,
    keyIds,
    outcomes,
    cursorTime: params.cursor?.time ?? null,
    cursorRequestId: params.cursor?.requestId ?? null,
    sorts,
  };
}
