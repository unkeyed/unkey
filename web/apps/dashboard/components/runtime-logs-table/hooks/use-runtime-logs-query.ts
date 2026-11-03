"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/logs/hooks/use-runtime-logs-filters";
import { useLoadMoreList } from "@/hooks/use-load-more-list";
import { PAGINATED_LIST_QUERY_OPTIONS } from "@/hooks/use-paginated-list-query";
import { getTimestampFromRelative } from "@/lib/duration";
import {
  type RuntimeLogsFilterValue,
  parseRuntimeLogsAttributeMatch,
} from "@/lib/schemas/runtime-logs.filter.schema";
import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import { trpcClient } from "@/lib/trpc/client";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { DEFAULT_LOGS_SINCE } from "@/lib/utils";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { type RuntimeLogRow, attachRowKeys, getLogKey } from "../utils/get-row-class";

type UseRuntimeLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
  // Incremented by the refresh control. A change re-anchors the query window so
  // the user sees logs that arrived since the last anchor.
  refreshNonce?: number;
};

const REALTIME_DATA_LIMIT = 100;

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
  const queryClient = trpc.useUtils();

  const resetKey = useMemo(
    () =>
      `${filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|")}|r:${refreshNonce}`,
    [filters, refreshNonce],
  );

  // Pinned upper bound of the historical window. Offset pagination over a live
  // time series only stays stable if the window does not slide between page
  // navigations, so we anchor it here and only re-anchor on filter/refresh.
  const [queryTime, setQueryTime] = useState(() => Date.now());

  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, RuntimeLog>());

  const activeRealtimeLogsMap = useMemo(() => {
    return startPolling ? realtimeLogsMap : new Map<string, RuntimeLog>();
  }, [startPolling, realtimeLogsMap]);

  const realtimeLogs = useMemo(() => {
    return attachRowKeys(sortLogs(Array.from(activeRealtimeLogsMap.values())));
  }, [activeRealtimeLogsMap]);

  // Resolve the historical window once per filter/anchor change. Relative `since`
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
      // Total comes from the window-keyed query below; skip the per-page count(*).
      includeTotal: false,
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
    setRealtimeLogsMap(new Map());
  }

  // The total is the same for every page in the current window, so fetch it once
  // (page 1, limit 1) and reuse it instead of counting per page. Re-runs when the
  // window changes. Disabled in live mode (footer hidden).
  const totalQueryInput = useMemo(
    () => ({ ...queryInput, page: 1, limit: 1, includeTotal: true }),
    [queryInput],
  );
  const totalQuery = trpc.deploy.runtimeLogs.query.useQuery(totalQueryInput, {
    ...PAGINATED_LIST_QUERY_OPTIONS,
    enabled: !startPolling,
  });
  const total = totalQuery.isPreviousData ? null : (totalQuery.data?.total ?? null);

  const list = useLoadMoreList({
    queryKey: getQueryKey(trpc.deploy.runtimeLogs.query, queryInput, "infinite"),
    fetchPage: (page, signal) =>
      trpcClient.deploy.runtimeLogs.query.query({ ...queryInput, page }, { signal }),
    rowsOf,
    mapRows: attachRowKeys,
    total,
    pageSize: limit,
    paused: startPolling,
  });
  const historicalLogs = list.table.rows;
  const historicalKeys = useMemo(() => new Set(historicalLogs.map(getLogKey)), [historicalLogs]);

  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.deploy.runtimeLogs.query.fetch({
        ...queryInput,
        page: 1,
        // The poll only reads `result.logs`, so skip the count(*) it would discard.
        includeTotal: false,
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
          const key = getLogKey(log);
          // Skip if it already exists in either buffer.
          if (newMap.has(key) || historicalKeys.has(key)) {
            continue;
          }

          newMap.set(key, log);
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
      console.error("Error polling for new runtime logs:", error);
    }
  }, [
    queryInput,
    queryClient,
    limit,
    pollIntervalMs,
    historicalKeys,
    realtimeLogs,
    historicalLogs,
  ]);

  // pollForNewLogs changes identity on every successful poll (its deps include
  // realtimeLogs/historicalLogs). If the interval depended on it directly, each
  // batch of new logs would tear down and restart the timer, making the cadence
  // "5s after each response" rather than a fixed 5s — drift that compounds under
  // slow ClickHouse or network. Hold the latest callback in a ref so a single
  // stable interval can call it without re-subscribing.
  const pollForNewLogsRef = useRef(pollForNewLogs);
  useEffect(() => {
    pollForNewLogsRef.current = pollForNewLogs;
  }, [pollForNewLogs]);

  useEffect(() => {
    if (!startPolling) {
      return;
    }
    // Skip a tick while the previous poll is still in flight. ClickHouse allows
    // queries up to 20s, well past the 5s interval, so without this guard slow
    // responses would pile up overlapping requests against a struggling backend.
    let inFlight = false;
    const interval = setInterval(async () => {
      if (inFlight) {
        return;
      }
      inFlight = true;
      try {
        await pollForNewLogsRef.current();
      } finally {
        inFlight = false;
      }
    }, pollIntervalMs);
    return () => clearInterval(interval);
  }, [startPolling, pollIntervalMs]);

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

const sortLogs = (logs: RuntimeLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
