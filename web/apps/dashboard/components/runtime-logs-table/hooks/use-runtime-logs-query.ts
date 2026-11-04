"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/logs/hooks/use-runtime-logs-filters";
import { useLiveTail } from "@/hooks/use-live-tail";
import { useLoadMoreList } from "@/hooks/use-load-more-list";
import { getTimestampFromRelative } from "@/lib/duration";
import {
  type RuntimeLogsFilterValue,
  parseRuntimeLogsAttributeMatch,
} from "@/lib/schemas/runtime-logs.filter.schema";
import { trpcClient } from "@/lib/trpc/client";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { DEFAULT_LOGS_SINCE } from "@/lib/utils";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import { useParams } from "next/navigation";
import { useMemo, useState } from "react";
import { type RuntimeLogRow, attachRowKeys, getLogKey } from "../utils/get-row-class";

type UseRuntimeLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
  // Incremented by the refresh control. A change re-anchors the query window so
  // the user sees logs that arrived since the last anchor.
  refreshNonce?: number;
};

type RuntimeLogsPage = inferRouterOutputs<Router>["deploy"]["runtimeLogs"]["query"];

const rowsOf = (response: RuntimeLogsPage): RuntimeLogRow[] => response.logs;

function toAttributesQuery(
  filter: RuntimeLogsFilterValue | undefined,
):
  | { operator: "contains"; value: string }
  | { operator: "is"; path: string; value: string }
  | null {
  if (!filter) {
    return null;
  }
  if (filter.operator === "contains") {
    return { operator: "contains", value: String(filter.value) };
  }

  const match = parseRuntimeLogsAttributeMatch(String(filter.value));
  return match ? { operator: "is", ...match } : null;
}

export function useRuntimeLogsQuery({
  limit = 50,
  startPolling = false,
  pollIntervalMs = 5000,
  refreshNonce = 0,
}: UseRuntimeLogsQueryParams = {}) {
  const params = useParams<{ projectId: string }>();
  const { filters } = useRuntimeLogsFilters();

  const resetKey = useMemo(
    () =>
      `${filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|")}|r:${refreshNonce}`,
    [filters, refreshNonce],
  );

  // Pinned upper bound of the historical window. Offset pagination over a live
  // time series only stays stable if the window does not slide between page
  // navigations, so we anchor it here and only re-anchor on filter/refresh.
  const [queryTime, setQueryTime] = useState(() => Date.now());

  // filters resolve their `startTime` against wall-clock-at-recompute (this memo
  // only re-runs on filter/anchor change, so it stays stable across page
  // navigations); their `endTime` is the pinned anchor. Explicit start/end are
  // used as-is. We always pass concrete start/end (not `since`) so the server
  // uses our window instead of resolving `endTime` to its own `Date.now()` per
  // request.
  const timeWindow = useMemo(() => {
    const startTimeFilter = filters.find((f) => f.field === "startTime");
    const endTimeFilter = filters.find((f) => f.field === "endTime");
    const sinceFilter = filters.find((f) => f.field === "since");

    if (sinceFilter) {
      return {
        startTime: getTimestampFromRelative(String(sinceFilter.value)),
        endTime: queryTime,
      };
    }
    if (startTimeFilter && endTimeFilter) {
      return {
        startTime: Number(startTimeFilter.value),
        endTime: Number(endTimeFilter.value),
      };
    }
    return {
      startTime: getTimestampFromRelative(DEFAULT_LOGS_SINCE),
      endTime: queryTime,
    };
  }, [filters, queryTime]);

  const queryInput = useMemo(() => {
    const severityFilters = filters
      .filter((f) => f.field === "severity")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));
    const regionFilters = filters
      .filter((f) => f.field === "region")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));
    const instanceIdFilters = filters
      .filter((f) => f.field === "instanceId")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));
    const environmentIdFilters = filters
      .filter((f) => f.field === "environmentId")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));
    const deploymentIdFilters = filters
      .filter((f) => f.field === "deploymentId")
      .map((f) => String(f.value))
      .filter(Boolean);
    const appIdFilters = filters
      .filter((f) => f.field === "appId")
      .map((f) => String(f.value))
      .filter(Boolean);
    const messageFilter = filters.find((f) => f.field === "message");
    const attributesFilter = filters.find((f) => f.field === "attributes");

    return {
      projectId: params.projectId,
      appId: appIdFilters,
      deploymentId: deploymentIdFilters,
      limit,
      startTime: timeWindow.startTime,
      endTime: timeWindow.endTime,
      since: null,
      severity: severityFilters.length > 0 ? { filters: severityFilters } : null,
      region: regionFilters.length > 0 ? { filters: regionFilters } : null,
      message: messageFilter ? String(messageFilter.value) : null,
      attributes: toAttributesQuery(attributesFilter),
      instanceId: instanceIdFilters.length > 0 ? { filters: instanceIdFilters } : null,
      environmentId: environmentIdFilters.length > 0 ? { filters: environmentIdFilters } : null,
    };
  }, [filters, limit, params.projectId, timeWindow]);

  const [prevResetKey, setPrevResetKey] = useState(resetKey);
  if (prevResetKey !== resetKey) {
    setPrevResetKey(resetKey);
    setQueryTime(Date.now());
  }

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.deploy.runtimeLogs.query, queryInput, "infinite"),
    fetchPage: (page, signal) =>
      trpcClient.deploy.runtimeLogs.query.query({ ...queryInput, page }, { signal }),
    rowsOf,
    mapRows: attachRowKeys,
    pageSize: limit,
  });
  const historicalLogs = list.table.rows;
  const liveLogs = useLiveTail({
    enabled: startPolling,
    intervalMs: pollIntervalMs,
    queryKey: getQueryKey(trpc.deploy.runtimeLogs.query, queryInput),
    windowStart: queryInput.startTime,
    fetchNewest: async (window, signal) => {
      const page = await trpcClient.deploy.runtimeLogs.query.query(
        { ...queryInput, ...window, page: 1 },
        { signal },
      );
      return page.logs;
    },
    keyOf: getLogKey,
    historical: historicalLogs,
  });
  const realtimeLogs = useMemo(() => attachRowKeys(liveLogs), [liveLogs]);

  return {
    realtimeLogs,
    table: list.table,
    totalCount: list.totalCount,
  };
}
