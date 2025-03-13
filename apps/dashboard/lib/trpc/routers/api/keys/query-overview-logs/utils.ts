import type { KeysQueryOverviewLogsPayload } from "@/app/(app)/apis/[apiId]/_overview/components/table/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { KeysOverviewLogsParams } from "@unkey/clickhouse/src/keys/keys";

export function transformKeysFilters(
  params: KeysQueryOverviewLogsPayload,
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

  const names =
    params.names?.map((k) => ({
      operator: k.operator,
      value: k.value,
    })) ?? null;

  const identities =
    params.identities?.map((k) => ({
      operator: k.operator,
      value: k.value,
    })) ?? null;

  const outcomes =
    params.outcomes?.map((o) => ({
      operator: "is" as const,
      value: o.value,
    })) ?? null;

  return {
    limit: params.limit,
    startTime,
    endTime,
    keyIds,
    names,
    identities,
    outcomes,
    cursorTime: params.cursor?.time ?? null,
    cursorRequestId: params.cursor?.requestId ?? null,
  };
}
