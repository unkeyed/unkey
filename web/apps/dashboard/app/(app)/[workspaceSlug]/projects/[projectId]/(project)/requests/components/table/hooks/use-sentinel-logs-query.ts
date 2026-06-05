"use client";

import { useSentinelLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-sentinel-logs-filters";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { trpc } from "@/lib/trpc/client";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

type UseSentinelLogsQueryParams = {
  limit?: number;
  startPolling?: boolean;
  pollIntervalMs?: number;
};

const REALTIME_DATA_LIMIT = 100;

export function useSentinelLogsQuery({
  limit = 50,
  startPolling = false,
  pollIntervalMs = 2000,
}: UseSentinelLogsQueryParams = {}) {
  const { projectId } = useProjectData();
  const { filters } = useSentinelLogsFilters();
  const searchParams = useSearchParams();
  // Optional ?appId= narrows the project-wide view to a single app.
  const appId = searchParams.get("appId");
  const queryClient = trpc.useUtils();

  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, SentinelLogsResponse>(),
  );
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(
    () => new Map<string, SentinelLogsResponse>(),
  );

  const queryInput = useMemo(() => {
    // Extract filters
    const statusFilters = filters.filter((f) => f.field === "status").map((f) => Number(f.value));

    const methodFilters = filters.filter((f) => f.field === "methods").map((f) => String(f.value));

    const pathFilters = filters
      .filter((f) => f.field === "paths")
      .map((f) => ({
        operator: "contains" as const,
        value: String(f.value),
      }));

    const deploymentIdFilter = filters.find((f) => f.field === "deploymentId");
    const environmentIdFilters = filters
      .filter((f) => f.field === "environmentId")
      .map((f) => String(f.value));

    // Extract time filters
    const startTimeFilter = filters.find((f) => f.field === "startTime");
    const endTimeFilter = filters.find((f) => f.field === "endTime");
    const sinceFilter = filters.find((f) => f.field === "since");

    return {
      projectId,
      appId: appId ?? null,
      deploymentId: deploymentIdFilter ? String(deploymentIdFilter.value) : null,
      environmentId: environmentIdFilters,
      limit,
      startTime: startTimeFilter ? Number(startTimeFilter.value) : null,
      endTime: endTimeFilter ? Number(endTimeFilter.value) : null,
      since: sinceFilter ? String(sinceFilter.value) : null,
      statusCodes: statusFilters.length > 0 ? statusFilters : null,
      methods: methodFilters.length > 0 ? methodFilters : null,
      paths: pathFilters.length > 0 ? pathFilters : null,
    };
  }, [filters, limit, projectId, appId]);

  const { data, isLoading, error, hasNextPage, fetchNextPage, isFetchingNextPage } =
    trpc.deploy.sentinelLogs.query.useInfiniteQuery(queryInput, {
      getNextPageParam: (lastPage) => (lastPage.hasMore ? lastPage.nextCursor : undefined),
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    });

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(realtimeLogsMap.values()));
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const total = data?.pages[0]?.total ?? 0;

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.deploy.sentinelLogs.query.fetch({
        ...queryInput,
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
          // Skip if exists in either map
          if (newMap.has(log.request_id) || historicalLogsMap.has(log.request_id)) {
            continue;
          }

          newMap.set(log.request_id, log);
          added++;

          // Remove oldest entries when exceeding the size limit `100`
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

  // Set up polling effect
  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  // Update historical logs effect
  useEffect(() => {
    if (data) {
      const newMap = new Map<string, SentinelLogsResponse>();
      data.pages.forEach((page) => {
        page.logs.forEach((log) => {
          newMap.set(log.request_id, log);
        });
      });
      setHistoricalLogsMap(newMap);
    }
  }, [data]);

  // Reset realtime logs effect
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  return {
    realtimeLogs,
    historicalLogs,
    total,
    isLoading,
    error,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}

const sortLogs = (logs: SentinelLogsResponse[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
