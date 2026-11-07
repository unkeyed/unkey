import { keyDetailsFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useLiveTail } from "@/hooks/use-live-tail";
import { useLoadMoreList } from "@/hooks/use-load-more-list";
import { getTimestampFromRelative } from "@/lib/duration";
import { trpcClient } from "@/lib/trpc/client";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useMemo } from "react";
import type { KeyDetailsLogsPayload } from "../schema/query-logs.schema";

type KeyDetailsLogsPage = inferRouterOutputs<Router>["key"]["logs"]["query"];

const rowsOf = (response: KeyDetailsLogsPage) => response.logs;
const requestIdOf = (log: KeyDetailsLog) => log.request_id;

type UseKeyDetailsLogsQueryParams = {
  limit?: number;
  keyId: string;
  keyspaceId: string;
  pollIntervalMs?: number;
  startPolling?: boolean;
};

export function useKeyDetailsLogsQuery({
  keyId,
  keyspaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseKeyDetailsLogsQueryParams) {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: Omit<KeyDetailsLogsPayload, "page"> = {
      limit,
      keyId,
      keyspaceId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      outcomes: [],
      tags: [],
      since: "",
    };
    let since = "";

    filters.forEach((filter) => {
      const fieldConfig = keyDetailsFilterFieldConfig[filter.field];
      const validOperators = fieldConfig?.operators;
      if (!validOperators) {
        return;
      }

      switch (filter.field) {
        case "tags": {
          if (typeof filter.value === "string") {
            params.tags?.push({
              value: filter.value,
              operator: filter.operator as "is" | "contains" | "startsWith" | "endsWith",
            });
          }
          break;
        }
        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes?.push({
              value: filter.value as ValidOutcome,
              operator: "is",
            });
          }
          break;
        }
        case "startTime":
        case "endTime": {
          const numValue =
            typeof filter.value === "number"
              ? filter.value
              : typeof filter.value === "string"
                ? Number(filter.value)
                : Number.NaN;
          if (!Number.isNaN(numValue)) {
            params[filter.field] = numValue;
          }
          break;
        }
        case "since":
          if (typeof filter.value === "string") {
            since = filter.value;
          }
          break;
      }
    });

    if (since) {
      params.startTime = getTimestampFromRelative(since, timestamp);
      params.endTime = timestamp;
    }

    return params;
  }, [filters, limit, timestamp, keyId, keyspaceId]);

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.key.logs.query, queryParams, "infinite"),
    fetchPage: (page, signal) =>
      trpcClient.key.logs.query.query({ ...queryParams, page }, { signal }),
    rowsOf,
    idOf: requestIdOf,
    pageSize: limit,
  });
  const historicalLogs = list.table.rows;
  const realtimeLogs = useLiveTail({
    enabled: startPolling,
    intervalMs: pollIntervalMs,
    queryKey: getQueryKey(trpc.key.logs.query, queryParams),
    history: { startTime: queryParams.startTime, endTime: queryParams.endTime },
    fetchNewest: async (window, signal) => {
      const page = await trpcClient.key.logs.query.query(
        { ...queryParams, ...window, page: 1 },
        { signal },
      );
      return page.logs;
    },
    keyOf: requestIdOf,
    historical: historicalLogs,
  });

  return {
    realtimeLogs,
    table: list.table,
    totalCount: list.totalCount,
  };
}
