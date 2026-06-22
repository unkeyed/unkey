"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
} from "@/hooks/use-paginated-list-query";
import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import { trpc } from "@/lib/trpc/client";
import { DEFAULT_LOGS_SINCE, getTimestampFromRelative } from "@/lib/utils";
import { useParams, useSearchParams } from "next/navigation";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { attachRowKeys, getLogKey } from "../utils/get-row-class";

type UseRuntimeLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
  // Incremented by the refresh control. A change re-anchors the query window so
  // the user sees logs that arrived since the last anchor.
  refreshNonce?: number;
};

const REALTIME_DATA_LIMIT = 100;
const PREFETCH_PAGES_AHEAD = 2;

export function useRuntimeLogsQuery({
  limit = 50,
  startPolling = false,
  pollIntervalMs = 5000,
  refreshNonce = 0,
}: UseRuntimeLogsQueryParams = {}) {
  const params = useParams<{ projectId: string }>();
  const { filters } = useRuntimeLogsFilters();
  const searchParams = useSearchParams();
  // Optional ?appId= narrows the project-wide view to a single app.
  const appId = searchParams.get("appId");
  const queryClient = trpc.useUtils();

  const [page, setPage] = useQueryState("page", parseAsInteger.withDefault(1));
  const normalizedPage = Math.max(1, page);

  // Re-anchor the window, reset to page 1, and drop the realtime buffer whenever
  // the filters, active app, or refresh signal change. `appId` is included
  // because it also feeds `queryInput`: omitting it would let an app switch fire a
  // query against the stale page/window, flashing an empty page until the clamp
  // effect catches up.
  const resetKey = useMemo(
    () =>
      `${filters.map((f) => `${f.field}:${f.operator}:${f.value}`).join("|")}|app:${appId ?? ""}|r:${refreshNonce}`,
    [filters, appId, refreshNonce],
  );

  // A reset is pending when the key changed but the effect below hasn't committed
  // setPage(1) yet. We force page 1 synchronously here because the render that
  // observes the change still sees the old normalizedPage — feeding that to the
  // query would fire one stale request for the previous page against the new
  // window before setPage(1) commits.
  const prevResetKeyRef = useRef<string | null>(null);
  const isResetting = prevResetKeyRef.current !== null && resetKey !== prevResetKeyRef.current;

  // Live mode is a "tail the newest logs" view: it shows page 1 only and streams
  // inserts in via the realtime buffer — pagination is disabled (see the consumer,
  // which hides the footer). A pending reset also pins the effective page to 1.
  const effectivePage = startPolling || isResetting ? 1 : normalizedPage;

  // Pinned upper bound of the historical window. Offset pagination over a live
  // time series only stays stable if the window does not slide between page
  // navigations, so we anchor it here and only re-anchor on filter/refresh.
  const [queryTime, setQueryTime] = useState(() => Date.now());

  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, RuntimeLog>());

  // The realtime buffer is only surfaced in live mode (which is always page 1).
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
    const messageFilter = filters.find((f) => f.field === "message");

    return {
      projectId: params.projectId,
      appId: appId ?? null,
      deploymentId: deploymentIdFilters,
      limit,
      page: effectivePage,
      // Total comes from the window-keyed query below; skip the per-page count(*).
      includeTotal: false,
      startTime: timeWindow.startTime,
      endTime: timeWindow.endTime,
      since: null,
      severity: severityFilters.length > 0 ? { filters: severityFilters } : null,
      region: regionFilters.length > 0 ? { filters: regionFilters } : null,
      message: messageFilter ? String(messageFilter.value) : null,
      instanceId: instanceIdFilters.length > 0 ? { filters: instanceIdFilters } : null,
      environmentId: environmentIdFilters.length > 0 ? { filters: environmentIdFilters } : null,
    };
  }, [filters, limit, params.projectId, appId, effectivePage, timeWindow]);

  const { data, isLoading, error, isFetching } = trpc.deploy.runtimeLogs.query.useQuery(
    queryInput,
    PAGINATED_LIST_QUERY_OPTIONS,
  );

  // The total is the same for every page in the current window, so fetch it once
  // (page 1, limit 1) and reuse it instead of counting per page. Re-runs when the
  // window changes. Disabled in live mode (footer hidden).
  const totalQueryInput = useMemo(
    () => ({ ...queryInput, page: 1, limit: 1, includeTotal: true }),
    [queryInput],
  );
  const { data: totalData, isFetching: isTotalFetching } = trpc.deploy.runtimeLogs.query.useQuery(
    totalQueryInput,
    {
      ...PAGINATED_LIST_QUERY_OPTIONS,
      enabled: !startPolling,
    },
  );

  // Keyed page rows; the live poll dedups incoming rows against this map.
  const historicalLogsMap = useMemo(() => {
    const map = new Map<string, RuntimeLog>();
    if (data) {
      data.logs.forEach((log) => {
        map.set(getLogKey(log), log);
      });
    }
    return map;
  }, [data]);

  // Server page verbatim (no content dedup); attachRowKeys only makes React keys unique.
  const historicalLogs = useMemo(() => attachRowKeys(data?.logs ?? []), [data]);

  const totalCount = totalData?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / limit));

  // Commit the reset signalled synchronously above: advance the ref, snap to
  // page 1, re-anchor the window, and drop the realtime buffer.
  useEffect(() => {
    if (prevResetKeyRef.current === null) {
      prevResetKeyRef.current = resetKey;
      return;
    }
    if (resetKey !== prevResetKeyRef.current) {
      prevResetKeyRef.current = resetKey;
      setPage(1);
      setQueryTime(Date.now());
      setRealtimeLogsMap(new Map());
    }
  }, [resetKey, setPage]);

  // Entering live mode pins the view to page 1; clear any stale page param so
  // leaving live mode doesn't drop the user on a page that no longer makes sense.
  useEffect(() => {
    if (startPolling && page !== 1) {
      setPage(1);
    }
  }, [startPolling, page, setPage]);

  // Clamp page once the count resolves (off while live). The `totalData == null`
  // guard stops a deep-linked ?page=3 from snapping to 1 before the count returns.
  useEffect(() => {
    if (startPolling || totalData == null) {
      return;
    }
    if (effectivePage > totalPages) {
      setPage(totalPages);
    }
  }, [startPolling, totalData, effectivePage, totalPages, setPage]);

  // Prefetch the next few pages for snappy navigation (skipped while live).
  useEffect(() => {
    if (startPolling) {
      return;
    }
    for (let i = 1; i <= PREFETCH_PAGES_AHEAD; i++) {
      const nextPage = effectivePage + i;
      if (nextPage > totalPages) {
        break;
      }
      queryClient.deploy.runtimeLogs.query.prefetch(
        { ...queryInput, page: nextPage },
        PAGINATED_LIST_PREFETCH_OPTIONS,
      );
    }
  }, [startPolling, effectivePage, totalPages, queryInput, queryClient.deploy.runtimeLogs.query]);

  // Poll for new logs (page 1 only).
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
          if (newMap.has(key) || historicalLogsMap.has(key)) {
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
    historicalLogsMap,
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

  const onPageChange = useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );

  const isInitialLoading = isLoading && !data;
  const isNavigating = isFetching && !isInitialLoading;

  // Hold the footer in its loading state until the separate count first resolves,
  // so it doesn't flash out when rows arrive before the total. `isFetching` (not
  // `=== undefined`) so an errored/disabled count reports done, not stuck.
  const isCountLoading = isTotalFetching && totalData === undefined;

  return {
    realtimeLogs,
    historicalLogs,
    totalCount,
    error,
    isLoading: isInitialLoading,
    isCountLoading,
    isNavigating,
    page: effectivePage,
    pageSize: limit,
    totalPages,
    onPageChange,
  };
}

const sortLogs = (logs: RuntimeLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
