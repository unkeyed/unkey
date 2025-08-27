import type { AuditQueryLogsPayload } from "@/app/(app)/[workspace]/audit/components/table/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { AuditQueryLogsParams } from "./schema";

export function transformFilters(
  params: AuditQueryLogsPayload,
): Omit<AuditQueryLogsParams, "workspaceId"> {
  let startTime = params.startTime;
  let endTime = params.endTime;

  if (params.since) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  const events =
    params.events?.filters.map((f) => ({
      operator: f.operator,
      value: f.value,
    })) ?? [];

  const users =
    params.users?.filters.map((f) => ({
      operator: f.operator,
      value: f.value,
    })) ?? [];

  const rootKeys =
    params.rootKeys?.filters.map((f) => ({
      operator: f.operator,
      value: f.value,
    })) ?? [];

  return {
    bucket: params.bucket,
    events,
    users,
    rootKeys,
    startTime,
    endTime,
    limit: params.limit,
    cursor: params.cursor,
  };
}
