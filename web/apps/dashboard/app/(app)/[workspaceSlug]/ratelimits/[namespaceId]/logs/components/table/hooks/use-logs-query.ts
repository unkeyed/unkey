import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import type { RatelimitLog, RatelimitLogEnrichment } from "@unkey/clickhouse/src/ratelimits";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useFilters } from "../../../hooks/use-filters";
import type { RatelimitQueryLogsPayload } from "../query-logs.schema";

export type EnrichedRatelimitLog = RatelimitLog & RatelimitLogEnrichment & { isEnriched: boolean };

const ENRICHMENT_DEFAULTS: Omit<RatelimitLogEnrichment, "request_id"> = {
  host: "",
  method: "",
  path: "",
  request_headers: [],
  request_body: "",
  response_status: 0,
  response_headers: [],
  response_body: "",
  service_latency: 0,
  user_agent: "",
  region: "",
};

function enrichLogs(
  logs: RatelimitLog[],
  enrichmentMap: Map<string, RatelimitLogEnrichment>,
): EnrichedRatelimitLog[] {
  return logs.map((log) => {
    const enrichment = enrichmentMap.get(log.request_id);
    return {
      ...ENRICHMENT_DEFAULTS,
      ...log,
      ...(enrichment ?? {}),
      isEnriched: enrichment !== undefined,
    };
  });
}

type UseLogsQueryParams = {
  limit?: number;
  pollIntervalMs?: number;
  startPolling?: boolean;
  namespaceId: string;
};

const REALTIME_DATA_LIMIT = 100;
export function useRatelimitLogsQuery({
  namespaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseLogsQueryParams) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(
    () => new Map<string, EnrichedRatelimitLog>(),
  );
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(
    () => new Map<string, EnrichedRatelimitLog>(),
  );
  const enrichmentMapRef = useRef(new Map<string, RatelimitLogEnrichment>());
  const [enrichmentVersion, setEnrichmentVersion] = useState(0);
  const [totalCount, setTotalCount] = useState(0);

  const { queryTime: timestamp } = useQueryTime();

  const { filters } = useFilters();
  const queryClient = trpc.useUtils();

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(realtimeLogsMap.values()));
  }, [realtimeLogsMap]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const queryParams = useMemo(() => {
    const params: RatelimitQueryLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      requestIds: { filters: [] },
      identifiers: { filters: [] },
      status: { filters: [] },
      namespaceId,
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "identifiers": {
          if (typeof filter.value !== "string") {
            console.error("Identifiers filter value type has to be 'string'");
            return;
          }
          params.identifiers?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
        case "status": {
          if (typeof filter.value !== "string") {
            console.error("Status filter value type has to be 'string'");
            return;
          }
          params.status?.filters.push({
            operator: "is",
            value: filter.value as "blocked" | "passed",
          });
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'string'`);
            return;
          }
          params[filter.field] = filter.value;
          break;
        }

        case "since": {
          if (typeof filter.value !== "string") {
            console.error("Since filter value type has to be 'string'");
            return;
          }
          params.since = filter.value;
          break;
        }
      }
    });

    return params;
  }, [filters, limit, namespaceId, timestamp]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.ratelimit.logs.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Fetch enrichment data for a batch of logs
  const fetchEnrichment = useCallback(
    async (logs: RatelimitLog[]) => {
      if (logs.length === 0) {
        return;
      }

      const unenrichedIds = logs
        .filter((log) => !enrichmentMapRef.current.has(log.request_id))
        .map((log) => log.request_id);

      if (unenrichedIds.length === 0) {
        return;
      }

      const times = logs.map((l) => l.time);
      const minTime = Math.min(...times);
      const maxTime = Math.max(...times);

      // Batch into chunks of 100 to stay within the tRPC endpoint limit
      const BATCH_SIZE = 100;
      const chunks: string[][] = [];
      for (let i = 0; i < unenrichedIds.length; i += BATCH_SIZE) {
        chunks.push(unenrichedIds.slice(i, i + BATCH_SIZE));
      }

      try {
        const results = await Promise.all(
          chunks.map((chunk) =>
            queryClient.ratelimit.logs.enrichment.fetch({
              requestIds: chunk,
              startTime: minTime,
              endTime: maxTime,
            }),
          ),
        );

        let added = false;
        for (const result of results) {
          for (const item of result.enrichment) {
            enrichmentMapRef.current.set(item.request_id, item);
            added = true;
          }
        }

        if (added) {
          setEnrichmentVersion((v) => v + 1);
        }
      } catch (error) {
        console.error("Error fetching log enrichment:", error);
      }
    },
    [queryClient],
  );

  // Query for new logs (polling)
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.ratelimit.logs.query.fetch({
        ...queryParams,
        startTime: latestTime ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
      });

      if (result.ratelimitLogs.length === 0) {
        return;
      }

      // Build the list of new logs outside the updater to avoid mutation inside React setState
      const newLogs: RatelimitLog[] = [];
      for (const log of result.ratelimitLogs) {
        if (realtimeLogsMap.has(log.request_id) || historicalLogsMap.has(log.request_id)) {
          continue;
        }
        newLogs.push(log);
      }

      if (newLogs.length > 0) {
        setRealtimeLogsMap((prevMap) => {
          const newMap = new Map(prevMap);

          for (const log of newLogs) {
            if (newMap.has(log.request_id)) {
              continue;
            }

            const enrichment = enrichmentMapRef.current.get(log.request_id);
            const enriched: EnrichedRatelimitLog = {
              ...ENRICHMENT_DEFAULTS,
              ...log,
              ...(enrichment ?? {}),
              isEnriched: enrichment !== undefined,
            };
            newMap.set(log.request_id, enriched);

            if (newMap.size > Math.min(limit, REALTIME_DATA_LIMIT)) {
              const entries = Array.from(newMap.entries());
              const oldestEntry = entries.reduce((oldest, current) => {
                return oldest[1].time < current[1].time ? oldest : current;
              });
              newMap.delete(oldestEntry[0]);
            }
          }

          return newMap;
        });

        // Fire enrichment for new logs in background
        fetchEnrichment(newLogs);
      }
    } catch (error) {
      console.error("Error polling for new logs:", error);
    }
  }, [
    queryParams,
    queryClient,
    limit,
    pollIntervalMs,
    historicalLogsMap,
    realtimeLogsMap,
    realtimeLogs,
    historicalLogs,
    fetchEnrichment,
  ]);

  // Set up polling effect
  useEffect(() => {
    if (startPolling) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, pollForNewLogs, pollIntervalMs]);

  // Fetch enrichment when new initial data arrives
  useEffect(() => {
    if (initialData) {
      const allLogs: RatelimitLog[] = [];
      initialData.pages.forEach((page) => {
        for (const log of page.ratelimitLogs) {
          allLogs.push(log);
        }
      });

      if (initialData.pages.length > 0) {
        setTotalCount(initialData.pages[0].total);
      }

      fetchEnrichment(allLogs);
    }
  }, [initialData, fetchEnrichment]);

  // Re-merge enrichment into historical and realtime logs when enrichment changes
  // biome-ignore lint/correctness/useExhaustiveDependencies: enrichmentVersion triggers re-merge when ref updates
  useEffect(() => {
    if (initialData) {
      const allLogs: RatelimitLog[] = [];
      initialData.pages.forEach((page) => {
        for (const log of page.ratelimitLogs) {
          allLogs.push(log);
        }
      });

      const newMap = new Map<string, EnrichedRatelimitLog>();
      for (const log of enrichLogs(allLogs, enrichmentMapRef.current)) {
        newMap.set(log.request_id, log);
      }
      setHistoricalLogsMap(newMap);
    }

    // Also re-merge enrichment into realtime logs
    setRealtimeLogsMap((prevMap) => {
      if (prevMap.size === 0) {
        return prevMap;
      }
      const newMap = new Map<string, EnrichedRatelimitLog>();
      let changed = false;
      for (const [id, log] of prevMap) {
        const enrichment = enrichmentMapRef.current.get(id);
        if (enrichment && !log.isEnriched) {
          newMap.set(id, { ...log, ...enrichment, isEnriched: true });
          changed = true;
        } else {
          newMap.set(id, log);
        }
      }
      return changed ? newMap : prevMap;
    });
  }, [initialData, enrichmentVersion]);

  // Clear enrichment cache when query params change (filters, time range, namespace)
  // biome-ignore lint/correctness/useExhaustiveDependencies: queryParams covers all filter/time/namespace changes
  useEffect(() => {
    enrichmentMapRef.current.clear();
    setEnrichmentVersion(0);
  }, [queryParams]);

  // Reset realtime logs effect
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  return {
    realtimeLogs,
    historicalLogs,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    isPolling: startPolling,
    totalCount,
  };
}

const sortLogs = (logs: EnrichedRatelimitLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
