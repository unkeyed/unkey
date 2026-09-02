"use client";

import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  computeTotalPages,
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  paginationFilterKey,
  usePaginatedNavigation,
  usePaginatedPage,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import { DEFAULT_LOGS_SINCE, getTimestampFromRelative } from "@/lib/utils";

type UseRequestLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
  // Incremented by the refresh control. A change re-anchors the query window so
  // the user sees logs that arrived since the last anchor.
  refreshNonce?: number;
};

const REALTIME_DATA_LIMIT = 100;

export function useRequestLogsQuery({
  limit = 50,
  startPolling = false,
  pollIntervalMs = 2000,
  refreshNonce = 0,
}: UseRequestLogsQueryParams = {}) {
  const { projectId } = useProjectData();
  const { filters } = useRequestLogsFilters();
  const queryClient = trpc.useUtils();

  // A change of the filters or of the refresh signal resets the page, the window
  // anchor, and the realtime buffer.
  const resetKey = useMemo(
    () => `${paginationFilterKey(filters)}|r:${refreshNonce}`,
    [filters, refreshNonce],
  );

  // usePaginatedPage owns the URL page and the synchronous reset-to-1 on a
  // resetKey change, so `page` is already 1 on the render that first observes
  // one — no stale request for the previous page against the new window.
  const { page, setPage } = usePaginatedPage(resetKey);

  // Live mode is a "tail the newest logs" view: it shows page 1 only and streams
  // inserts in via the realtime buffer — pagination is disabled (see the consumer,
  // which hides the footer).
  const effectivePage = startPolling ? 1 : page;

  // Pinned upper bound of the historical window. Offset pagination over a live
  // time series only stays stable if the window does not slide between page
  // navigations, so we anchor it here and only re-anchor on filter/refresh.
  const [queryTime, setQueryTime] = useState(() => Date.now());

  const [realtimeLogsMap, setRealtimeLogsMap] = useState(
    () => new Map<string, RequestLogsResponse>(),
  );

  // The realtime buffer is only surfaced in live mode (which is always page 1).
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
      page: effectivePage,
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
  }, [filters, limit, projectId, effectivePage, timeWindow]);

  const { data, isLoading, error, isFetching } = trpc.deploy.requestLogs.query.useQuery(
    queryInput,
    PAGINATED_LIST_QUERY_OPTIONS,
  );

  const historicalLogsMap = useMemo(() => {
    const map = new Map<string, RequestLogsResponse>();
    if (data) {
      data.logs.forEach((log) => {
        map.set(log.request_id, log);
      });
    }
    return map;
  }, [data]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const totalCount = data?.total ?? 0;
  const totalPages = computeTotalPages(totalCount, limit);

  // Feature-specific half of the reset: re-anchor the window and drop the
  // realtime buffer. usePaginatedPage owns the page half off the same resetKey;
  // both fire in the same commit. Skips the mount pass so a first-load
  // deep link keeps its anchor.
  const prevResetKeyRef = useRef<string | null>(null);
  useEffect(() => {
    if (prevResetKeyRef.current === null) {
      prevResetKeyRef.current = resetKey;
      return;
    }
    if (resetKey !== prevResetKeyRef.current) {
      prevResetKeyRef.current = resetKey;
      setQueryTime(Date.now());
      setRealtimeLogsMap(new Map());
    }
  }, [resetKey]);

  // Entering live mode pins the view to page 1; clear any stale page param so
  // leaving live mode doesn't drop the user on a page that no longer makes sense.
  useEffect(() => {
    if (startPolling && page !== 1) {
      setPage(1);
    }
  }, [startPolling, page, setPage]);

  // `enabled: !startPolling` suspends the clamp and the adjacent-page prefetch
  // while live, where the view is pinned to page 1 and the footer is hidden.
  const { onPageChange, isInitialLoading, isNavigating } = usePaginatedNavigation({
    data,
    page: effectivePage,
    totalPages,
    setPage,
    isLoading,
    isFetching,
    queryParams: queryInput,
    prefetch: (params) =>
      queryClient.deploy.requestLogs.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
    enabled: !startPolling,
  });

  // Poll for new logs (page 1 only).
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
          if (newMap.has(log.request_id) || historicalLogsMap.has(log.request_id)) {
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
  }, [
    queryInput,
    queryClient,
    limit,
    pollIntervalMs,
    historicalLogsMap,
    realtimeLogs,
    historicalLogs,
  ]);

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
    historicalLogs,
    totalCount,
    error,
    isLoading: isInitialLoading,
    isNavigating,
    page: effectivePage,
    pageSize: limit,
    totalPages,
    onPageChange,
  };
}

const sortLogs = (logs: RequestLogsResponse[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
