"use client";

import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/requests/hooks/use-request-logs-filters";
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
import { useCallback, useEffect, useMemo, useState } from "react";

type UseRequestLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
  // Incremented by the refresh control. A change re-anchors the query window so
  // the user sees logs that arrived since the last anchor.
  refreshNonce?: number;
};

const REALTIME_DATA_LIMIT = 100;

type RequestLogsPage = inferRouterOutputs<Router>["deploy"]["requestLogs"]["query"];

const rowsOf = (response: RequestLogsPage) => response.logs;
const totalOf = (response: RequestLogsPage) => response.total;
const requestIdOf = (log: RequestLogsResponse) => log.request_id;

export function useRequestLogsQuery({
  limit = 50,
  startPolling = false,
  pollIntervalMs = 2000,
  refreshNonce = 0,
}: UseRequestLogsQueryParams = {}) {
  const { projectId } = useDeployScope();
  const { filters } = useRequestLogsFilters();
  const queryClient = trpc.useUtils();

  const resetKey = useMemo(
    () => `${paginationFilterKey(filters)}|r:${refreshNonce}`,
    [filters, refreshNonce],
  );

  // Pinned upper bound of the historical window. Offset pagination over a live
  // time series only stays stable if the window does not slide between page
  // navigations, so we anchor it here and only re-anchor on filter/refresh.
  const [queryTime, setQueryTime] = useState(() => Date.now());

  const [realtimeLogsMap, setRealtimeLogsMap] = useState(
    () => new Map<string, RequestLogsResponse>(),
  );

  const activeRealtimeLogsMap = useMemo(() => {
    return startPolling ? realtimeLogsMap : new Map<string, RequestLogsResponse>();
  }, [startPolling, realtimeLogsMap]);

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(activeRealtimeLogsMap.values()));
  }, [activeRealtimeLogsMap]);

  // Resolve the historical window once per filter/anchor change. Relative `since`
  // filters resolve against the pinned anchor; explicit start/end are used as-is.
  // We always pass concrete start/end (not `since`) so the server uses our pinned
  // window instead of resolving `endTime` to its own `Date.now()` per request.
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
    setRealtimeLogsMap(new Map());
  }

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.deploy.requestLogs.query, queryInput, "infinite"),
    fetchPage: (page, signal) =>
      trpcClient.deploy.requestLogs.query.query({ ...queryInput, page }, { signal }),
    rowsOf,
    idOf: requestIdOf,
    totalOf,
    pageSize: limit,
    paused: startPolling,
  });
  const historicalLogs = list.table.rows;
  const historicalIds = useMemo(() => new Set(historicalLogs.map(requestIdOf)), [historicalLogs]);

  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.deploy.requestLogs.query.fetch({
        ...queryInput,
        page: 1,
        startTime: latestTime ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
      });

      if (result.logs.length === 0) {
        return;
      }

      setRealtimeLogsMap((prevMap) => {
        const newMap = new Map(prevMap);
        let added = 0;

        for (const log of result.logs) {
          // Skip if it already exists in either buffer.
          if (newMap.has(log.request_id) || historicalIds.has(log.request_id)) {
            continue;
          }

          newMap.set(log.request_id, log);
          added++;

          // Drop the oldest entry once the buffer exceeds its cap.
          if (newMap.size > Math.min(limit, REALTIME_DATA_LIMIT)) {
            const entries = Array.from(newMap.entries());
            const oldestEntry = entries.reduce((oldest, current) => {
              return oldest[1].time < current[1].time ? oldest : current;
            });
            newMap.delete(oldestEntry[0]);
          }
        }

        return added > 0 ? newMap : prevMap;
      });
    } catch (error) {
      console.error("Error polling for new logs:", error);
    }
  }, [queryInput, queryClient, limit, pollIntervalMs, historicalIds, realtimeLogs, historicalLogs]);

  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  // Clear the realtime buffer whenever live mode is turned off.
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  return {
    realtimeLogs,
    table: list.table,
    totalCount: list.totalCount,
  };
}

const sortLogs = (logs: RequestLogsResponse[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
