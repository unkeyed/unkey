"use client";

import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/requests/hooks/use-request-logs-filters";
import { useLiveTail } from "@/hooks/use-live-tail";
import { useLoadMoreList } from "@/hooks/use-load-more-list";
import { paginationFilterKey } from "@/hooks/use-paginated-list-query";
import { useDeployScope } from "@/lib/deploy-data";
import { getTimestampFromRelative } from "@/lib/duration";
import { trpcClient } from "@/lib/trpc/client";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { DEFAULT_LOGS_SINCE } from "@/lib/utils";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import { useMemo, useState } from "react";

type UseRequestLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
  // Incremented by the refresh control. A change re-anchors the query window so
  // the user sees logs that arrived since the last anchor.
  refreshNonce?: number;
};

type RequestLogsPage = inferRouterOutputs<Router>["deploy"]["requestLogs"]["query"];

const rowsOf = (response: RequestLogsPage) => response.logs;
const requestIdOf = (log: RequestLogsResponse) => log.request_id;

export function useRequestLogsQuery({
  limit = 50,
  startPolling = false,
  pollIntervalMs = 2000,
  refreshNonce = 0,
}: UseRequestLogsQueryParams = {}) {
  const { projectId } = useDeployScope();
  const { filters } = useRequestLogsFilters();

  const resetKey = useMemo(
    () => `${paginationFilterKey(filters)}|r:${refreshNonce}`,
    [filters, refreshNonce],
  );

  // Pinned upper bound of the historical window. Offset pagination over a live
  // time series only stays stable if the window does not slide between page
  // navigations, so we anchor it here and only re-anchor on filter/refresh.
  const [queryTime, setQueryTime] = useState(() => Date.now());

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
    const statusFilters = filters.filter((f) => f.field === "status").map((f) => Number(f.value));
    const methodFilters = filters.filter((f) => f.field === "methods").map((f) => String(f.value));
    const pathFilters = filters
      .filter((f) => f.field === "paths")
      .map((f) => ({
        operator: f.operator,
        value: String(f.value),
      }));
    const valuesFor = (field: "host" | "requestId" | "region") =>
      filters.filter((filter) => filter.field === field).map((filter) => String(filter.value));

    const appIdFilters = filters
      .filter((f) => f.field === "appId")
      .map((f) => String(f.value))
      .filter(Boolean);
    const deploymentIdFilters = filters
      .filter((f) => f.field === "deploymentId")
      .map((f) => String(f.value))
      .filter(Boolean);
    const environmentIdFilters = filters
      .filter((f) => f.field === "environmentId")
      .map((f) => String(f.value));

    return {
      projectId,
      appId: appIdFilters,
      deploymentId: deploymentIdFilters,
      environmentId: environmentIdFilters,
      limit,
      startTime: timeWindow.startTime,
      endTime: timeWindow.endTime,
      since: null,
      statusCodes: statusFilters.length > 0 ? statusFilters : null,
      methods: methodFilters.length > 0 ? methodFilters : null,
      paths: pathFilters.length > 0 ? pathFilters : null,
      host: valuesFor("host"),
      requestId: valuesFor("requestId"),
      region: valuesFor("region"),
    };
  }, [filters, limit, projectId, timeWindow]);

  const [prevResetKey, setPrevResetKey] = useState(resetKey);
  if (prevResetKey !== resetKey) {
    setPrevResetKey(resetKey);
    setQueryTime(Date.now());
  }

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.deploy.requestLogs.query, queryInput, "infinite"),
    fetchPage: (page, signal) =>
      trpcClient.deploy.requestLogs.query.query({ ...queryInput, page }, { signal }),
    rowsOf,
    idOf: requestIdOf,
    pageSize: limit,
  });
  const historicalLogs = list.table.rows;
  const realtimeLogs = useLiveTail({
    enabled: startPolling,
    intervalMs: pollIntervalMs,
    queryKey: getQueryKey(trpc.deploy.requestLogs.query, queryInput),
    fetchNewest: async (signal) => {
      const page = await trpcClient.deploy.requestLogs.query.query(
        { ...queryInput, page: 1, endTime: Date.now() },
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
