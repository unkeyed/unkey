"use client";

import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import { trpc } from "@/lib/trpc/client";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useRuntimeLogs } from "../../../context/runtime-logs-provider";
import type { RuntimeLogsFilter } from "../../../types";

const LIVE_POLL_INTERVAL_MS = 5_000;
// Cap how many realtime rows we hold above the "Live" divider before
// dropping the oldest. Keeps the bucket bounded under sustained traffic.
const REALTIME_BUFFER_LIMIT = 100;

type UseRuntimeLogsQueryParams = {
  limit?: number;
  filters: RuntimeLogsFilter[];
};

const getLogKey = (log: RuntimeLog): string =>
  `${log.time}-${log.region}-${log.instance_id}-${log.message}`;

export function useRuntimeLogsQuery({ limit = 50, filters }: UseRuntimeLogsQueryParams) {
  const params = useParams<{ projectId: string }>();
  const { isLive } = useRuntimeLogs();
  const queryClient = trpc.useUtils();

  const [historicalLogsMap, setHistoricalLogsMap] = useState(() => new Map<string, RuntimeLog>());
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(() => new Map<string, RuntimeLog>());

  // Transform filters to tRPC input format
  const queryInput = useMemo(() => {
    const severityFilters = filters
      .filter((f) => f.field === "severity")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));

    const messageFilter = filters.find((f) => f.field === "message");
    const startTimeFilter = filters.find((f) => f.field === "startTime");
    const endTimeFilter = filters.find((f) => f.field === "endTime");
    const sinceFilter = filters.find((f) => f.field === "since");
    const environmentIdFilters = filters
      .filter((f) => f.field === "environmentId")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));
    const deploymentIdFilter = filters.find((f) => f.field === "deploymentId");
    const regionFilters = filters
      .filter((f) => f.field === "region")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));
    const instanceIdFilters = filters
      .filter((f) => f.field === "instanceId")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));

    return {
      projectId: params.projectId,
      deploymentId: deploymentIdFilter ? String(deploymentIdFilter.value) : null,
      limit,
      startTime: startTimeFilter ? Number(startTimeFilter.value) : Date.now() - 6 * 60 * 60 * 1000,
      endTime: endTimeFilter ? Number(endTimeFilter.value) : Date.now(),
      since: sinceFilter ? String(sinceFilter.value) : "6h",
      severity: severityFilters.length > 0 ? { filters: severityFilters } : null,
      region: regionFilters.length > 0 ? { filters: regionFilters } : null,
      message: messageFilter ? String(messageFilter.value) : null,
      instanceId: instanceIdFilters.length > 0 ? { filters: instanceIdFilters } : null,
      environmentId: environmentIdFilters.length > 0 ? { filters: environmentIdFilters } : null,
    };
  }, [filters, limit, params]);

  const { data, isLoading, error, hasNextPage, fetchNextPage, isFetchingNextPage } =
    trpc.deploy.runtimeLogs.query.useInfiniteQuery(queryInput, {
      getNextPageParam: (lastPage) => (lastPage.hasMore ? lastPage.nextCursor : undefined),
      refetchOnWindowFocus: false,
    });

  const realtimeLogs = useMemo(
    () => sortLogsDesc(Array.from(realtimeLogsMap.values())),
    [realtimeLogsMap],
  );
  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  // Poll for new logs above the latest known timestamp. Ranged on (latest, now]
  // so we don't re-fetch already-displayed rows. Limit is sized to the realtime
  // buffer (not the historical page size) so a single burst fills the bucket
  // in one round-trip; rows beyond the buffer cap would be evicted anyway, so
  // there's nothing to gain from paginating further.
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.deploy.runtimeLogs.query.fetch({
        ...queryInput,
        limit: REALTIME_BUFFER_LIMIT,
        startTime: latestTime ?? Date.now() - LIVE_POLL_INTERVAL_MS,
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
          if (newMap.has(key) || historicalLogsMap.has(key)) {
            continue;
          }
          newMap.set(key, log);
          added++;

          if (newMap.size > REALTIME_BUFFER_LIMIT) {
            const oldest = Array.from(newMap.entries()).reduce((a, b) =>
              a[1].time < b[1].time ? a : b,
            );
            newMap.delete(oldest[0]);
          }
        }

        return added > 0 ? newMap : prevMap;
      });
    } catch (err) {
      console.error("Error polling for new runtime logs:", err);
    }
  }, [queryInput, queryClient, historicalLogsMap, realtimeLogs, historicalLogs]);

  // Run the poll loop only while Live is on.
  useEffect(() => {
    if (!isLive) {
      return;
    }
    const interval = setInterval(pollForNewLogs, LIVE_POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [isLive, pollForNewLogs]);

  // Mirror infinite-query pages into the historical map so realtime polling
  // can dedup against the rendered set.
  useEffect(() => {
    if (!data) {
      return;
    }
    const newMap = new Map<string, RuntimeLog>();
    for (const page of data.pages) {
      for (const log of page.logs) {
        newMap.set(getLogKey(log), log);
      }
    }
    setHistoricalLogsMap(newMap);
  }, [data]);

  // Drop the realtime bucket whenever Live toggles or the query input changes.
  // Without resetting on filter changes, rows polled under the previous filters
  // would linger above the Live divider while pollForNewLogs starts fetching
  // with the new filters.
  // biome-ignore lint/correctness/useExhaustiveDependencies: isLive + queryInput are intentional triggers; body resets state, doesn't read them.
  useEffect(() => {
    setRealtimeLogsMap(new Map());
  }, [isLive, queryInput]);

  return {
    logs: historicalLogs,
    realtimeLogs,
    isLoading,
    error,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}

const sortLogsDesc = (logs: RuntimeLog[]) => logs.toSorted((a, b) => b.time - a.time);
