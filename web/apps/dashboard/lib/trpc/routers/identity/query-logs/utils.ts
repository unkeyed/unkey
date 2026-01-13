import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { getTimestampFromRelative } from "@/lib/utils";
import type { identityLogsPayload } from "./query-logs.schema";
import type { z } from "zod";

type IdentityLogsInput = z.infer<typeof identityLogsPayload>;

// This should match the ClickHouse identityLogsParams schema exactly
export interface IdentityLogsClickHouseParams {
  workspaceId: string;
  keyIds: string[];
  limit: number;
  startTime: number;
  endTime: number;
  tags: Array<{
    value: string;
    operator: "is" | "contains" | "startsWith" | "endsWith";
  }> | null;
  outcomes: Array<{
    value: (typeof KEY_VERIFICATION_OUTCOMES)[number];
    operator: "is";
  }> | null;
  cursorTime: number | null;
}

export function transformIdentityLogsFilters(
  input: IdentityLogsInput,
  workspaceId: string,
  keyIds: string[],
): IdentityLogsClickHouseParams {
  let startTime = input.startTime;
  let endTime = input.endTime;

  // Handle relative time ranges (like "1h", "24h", etc.)
  if (input.since) {
    startTime = getTimestampFromRelative(input.since);
    endTime = Date.now();
  }

  return {
    workspaceId,
    keyIds,
    limit: input.limit,
    startTime,
    endTime,
    tags: input.tags,
    outcomes: input.outcomes as Array<{
      value: (typeof KEY_VERIFICATION_OUTCOMES)[number];
      operator: "is";
    }> | null,
    cursorTime: input.cursor,
  };
}