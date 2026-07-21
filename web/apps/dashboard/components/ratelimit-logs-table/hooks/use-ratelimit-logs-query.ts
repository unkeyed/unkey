import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import {
  PAGINATED_LIST_PREFETCH_OPTIONS,
  PAGINATED_LIST_QUERY_OPTIONS,
  computeTotalPages,
  paginationFilterKey,
  usePaginatedNavigation,
  usePaginatedPage,
} from "@/hooks/use-paginated-list-query";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import type { SortingState } from "@tanstack/react-table";
import type {
  RatelimitLog,
  RatelimitLogEnrichment,
  RatelimitLogsSort,
} from "@unkey/clickhouse/src/ratelimits";
import { createParser, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { RatelimitQueryLogsPayload } from "../schema/query-logs.schema";

// Maps TanStack column ids (set in create-ratelimit-logs-columns) to the
// server's sort enum. Only columns backed by `ratelimits_raw_v2` are sortable;
// enrichment-derived columns are absent here and stay non-sortable.
const SORT_COLUMN_BY_ID: Record<string, RatelimitLogsSort["column"]> = {
  time: "time",
  identifier: "identifier",
  status: "status",
};

// Persists the single-column sort in the URL as `<columnId>.<asc|desc>`
// (e.g. `?sort=time.desc`) so sorted views are shareable. Unknown columns or
// malformed values parse to "no sort", keeping deep links tamper-tolerant, and
// clearing the sort drops the param entirely (clearOnDefault + eq).
const sortingParser = createParser<SortingState>({
  parse(value) {
    const [id, direction] = value.split(".");
    if (!id || !(id in SORT_COLUMN_BY_ID) || (direction !== "asc" && direction !== "desc")) {
      return null;
    }
    return [{ id, desc: direction === "desc" }];
  },
  serialize(value) {
    const [sort] = value;
    return sort ? `${sort.id}.${sort.desc ? "desc" : "asc"}` : "";
  },
  eq(a, b) {
    return a[0]?.id === b[0]?.id && a[0]?.desc === b[0]?.desc;
  },
})
  .withOptions({ clearOnDefault: true })
  .withDefault([]);

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

// Merge a raw log with its enrichment (if any) into the shape the table renders.
// Defaults fill the enrichment columns so cells can read fields unconditionally;
// `isEnriched` is what cells actually gate their skeleton on.
function toEnrichedLog(
  log: RatelimitLog,
  enrichment: RatelimitLogEnrichment | undefined,
): EnrichedRatelimitLog {
  return {
    ...ENRICHMENT_DEFAULTS,
    ...log,
    ...(enrichment ?? {}),
    isEnriched: enrichment !== undefined,
  };
}

function enrichLogs(
  logs: RatelimitLog[],
  enrichmentMap: Map<string, RatelimitLogEnrichment>,
): EnrichedRatelimitLog[] {
  return logs.map((log) => toEnrichedLog(log, enrichmentMap.get(log.request_id)));
}

// Maximum number of real-time logs to store
const REALTIME_DATA_LIMIT = 100;

// The enrichment row (from api_requests_raw_v2) for a ratelimit event is the
// request record, whose `time` is slightly later than the ratelimit-decision
// `time` we key off of. Using the logs' exact min/max as the query window
// therefore clips the boundary rows — most visibly the newest (first) log,
// which sits exactly at maxTime. `request_id IN (...)` is the real filter, so
// padding the window is correctness-safe and only relaxes partition pruning.
const ENRICHMENT_TIME_BUFFER_MS = 60_000;

type UseRatelimitLogsQueryParams = {
  limit?: number;
  pollIntervalMs?: number;
  startPolling?: boolean;
  namespaceId: string;
};

export function useRatelimitLogsQuery({
  namespaceId,
  limit = 50,
  pollIntervalMs = 5000,
  startPolling = false,
}: UseRatelimitLogsQueryParams) {
  const [realtimeLogsMap, setRealtimeLogsMap] = useState(
    () => new Map<string, EnrichedRatelimitLog>(),
  );
  const enrichmentMapRef = useRef(new Map<string, RatelimitLogEnrichment>());
  const [enrichmentVersion, setEnrichmentVersion] = useState(0);
  const lastNonEmptyPageRef = useRef(1);

  const { filters } = useFilters();
  const queryClient = trpc.useUtils();
  const { queryTime: timestamp } = useQueryTime();

  // Server-side sort state, persisted in the URL (`?sort=`). Single-column
  // (TanStack is configured for single sort), defaults to newest-first.
  const [sorting, setSorting] = useQueryState("sort", sortingParser);

  // Stable string key from filter content plus the query-time anchor. When
  // either changes we reset pagination and clear the realtime/enrichment
  // buffers, since cached rows no longer belong to the new result set.
  const filtersKey = useMemo(
    () => `${paginationFilterKey(filters)}|ts:${timestamp}`,
    [filters, timestamp],
  );

  // Sort changes also reset pagination (the current OFFSET is only meaningful
  // relative to the current ordering), but unlike a filter change the result
  // *set* is unchanged, so the realtime/enrichment buffers are left intact —
  // the buffer-flush effect below keys off filtersKey alone.
  const sortKey = useMemo(
    () => sorting.map((sort) => `${sort.id}:${sort.desc ? "desc" : "asc"}`).join("|"),
    [sorting],
  );

  const { page, setPage } = usePaginatedPage(`${filtersKey}|sort:${sortKey}`);

  // Live tail only operates on page 1 and pagination is disabled while live, so
  // force the queried page to 1 whenever polling is active. Together with the
  // reset-key transition in usePaginatedPage this keeps the query, realtime
  // gating, and URL in sync without firing a stale request for whatever page
  // was last viewed.
  const queryPage = startPolling ? 1 : page;

  // Real-time logs only apply on the first page while polling is active
  const activeRealtimeLogsMap = useMemo(() => {
    return startPolling && queryPage === 1
      ? realtimeLogsMap
      : new Map<string, EnrichedRatelimitLog>();
  }, [startPolling, queryPage, realtimeLogsMap]);

  const realtimeLogs = useMemo(() => {
    return sortLogs(Array.from(activeRealtimeLogsMap.values()));
  }, [activeRealtimeLogsMap]);

  // Live tail is always newest-first, so sorting only applies to historical
  // pages. Unknown column ids (enrichment-derived columns) are dropped.
  const sorts = useMemo<RatelimitLogsSort[] | null>(() => {
    if (startPolling || sorting.length === 0) {
      return null;
    }
    const resolved = sorting.flatMap((sort) => {
      const column = SORT_COLUMN_BY_ID[sort.id];
      if (!column) {
        return [];
      }
      return [{ column, direction: sort.desc ? ("desc" as const) : ("asc" as const) }];
    });
    return resolved.length > 0 ? resolved : null;
  }, [sorting, startPolling]);

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
      page: queryPage,
      sorts,
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
          if (filter.value !== "blocked" && filter.value !== "passed") {
            console.error("Status filter value has to be 'blocked' or 'passed'");
            return;
          }
          params.status?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'number'`);
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
  }, [filters, limit, namespaceId, timestamp, queryPage, sorts]);

  // Flush the realtime + enrichment buffers when filters or the time window
  // change — cached rows no longer belong to the new result set. The page reset
  // itself (synchronous queryPage plus the setPage(1) commit) is owned by
  // usePaginatedPage; this effect only handles the feature-specific buffers,
  // and deliberately ignores sort changes, which keep the result set intact.
  const prevFiltersKeyRef = useRef<string | null>(null);
  useEffect(() => {
    if (prevFiltersKeyRef.current === null) {
      prevFiltersKeyRef.current = filtersKey;
      return;
    }
    if (filtersKey !== prevFiltersKeyRef.current) {
      prevFiltersKeyRef.current = filtersKey;
      setRealtimeLogsMap(new Map());
      enrichmentMapRef.current.clear();
      setEnrichmentVersion(0);
      lastNonEmptyPageRef.current = 1;
    }
  }, [filtersKey]);

  // Main query for historical data (page-based)
  const {
    data: logData,
    isLoading,
    isFetching,
  } = trpc.ratelimit.logs.query.useQuery(queryParams, PAGINATED_LIST_QUERY_OPTIONS);

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
              startTime: minTime - ENRICHMENT_TIME_BUFFER_MS,
              endTime: maxTime + ENRICHMENT_TIME_BUFFER_MS,
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

  // Derive enriched historical logs from query data; re-derive when enrichment updates
  // biome-ignore lint/correctness/useExhaustiveDependencies: enrichmentVersion triggers re-merge when the ref updates
  const historicalLogsMap = useMemo(() => {
    const map = new Map<string, EnrichedRatelimitLog>();
    if (logData) {
      for (const log of enrichLogs(logData.ratelimitLogs, enrichmentMapRef.current)) {
        map.set(log.request_id, log);
      }
    }
    return map;
  }, [logData, enrichmentVersion]);

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  // When the ClickHouse count query fails the tRPC handler returns `total:
  // null`. Falling back to `hasMore = page is full` keeps pagination usable
  // (the previous infinite-query path derived `hasMore` the same way) instead
  // of collapsing to `totalPages = 1` and hiding the footer.
  const pageRowCount = logData?.ratelimitLogs.length ?? 0;
  const knownTotal = logData?.total ?? null;
  const totalCount =
    knownTotal !== null ? Math.max(0, knownTotal) : (queryPage - 1) * limit + pageRowCount;
  // Without a count, an out-of-range page is only detectable once it comes
  // back empty. Reporting the last page seen with rows as the total lets the
  // clamp snap back to the last real page (or to 1 for a stale deep link that
  // never saw data) instead of stranding the user on an empty page. The
  // isFetching gate keeps keepPreviousData from crediting the previous page's
  // rows to the page still being fetched.
  useEffect(() => {
    if (!isFetching && pageRowCount > 0) {
      lastNonEmptyPageRef.current = queryPage;
    }
  }, [isFetching, pageRowCount, queryPage]);
  const isEmptyPageBeyondFirst = logData != null && pageRowCount === 0 && queryPage > 1;
  const totalPages =
    knownTotal !== null
      ? computeTotalPages(totalCount, limit)
      : isEmptyPageBeyondFirst
        ? Math.min(lastNonEmptyPageRef.current, queryPage - 1)
        : queryPage + (pageRowCount >= limit ? 1 : 0);

  const { onPageChange } = usePaginatedNavigation({
    data: logData,
    page: queryPage,
    totalPages,
    setPage,
    queryParams,
    prefetch: (params) =>
      queryClient.ratelimit.logs.query.prefetch(params, PAGINATED_LIST_PREFETCH_OPTIONS),
    enabled: !startPolling,
  });

  // Fetch enrichment whenever new historical data arrives
  useEffect(() => {
    if (logData) {
      fetchEnrichment(logData.ratelimitLogs);
    }
  }, [logData, fetchEnrichment]);

  // Re-merge enrichment into realtime logs when enrichment changes
  // biome-ignore lint/correctness/useExhaustiveDependencies: enrichmentVersion triggers the re-merge when the ref updates
  useEffect(() => {
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
  }, [enrichmentVersion]);

  // Query for new logs (polling) — only on page 1
  const pollForNewLogs = useCallback(async () => {
    try {
      const latestTime = realtimeLogs[0]?.time ?? historicalLogs[0]?.time;
      const result = await queryClient.ratelimit.logs.query.fetch({
        ...queryParams,
        startTime: latestTime ?? Date.now() - pollIntervalMs,
        endTime: Date.now(),
        page: 1,
      });

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

            const enriched = toEnrichedLog(log, enrichmentMapRef.current.get(log.request_id));
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
      }

      // Enrich brand-new logs, and retry any realtime rows still missing
      // enrichment. The newest live-tail row routinely has no
      // api_requests_raw_v2 record at first poll: the request log lands a beat
      // after the ratelimit decision we key off of, so the initial fetch finds
      // nothing. Since that row is no longer "new" on the next poll, a
      // single fetch per batch leaves it un-enriched forever (empty
      // response_body, etc.) — the missing-data-on-first-row symptom. Re-passing
      // unenriched rows lets a later poll pick up the now-ingested record;
      // fetchEnrichment de-dupes against the cache, so already-enriched rows
      // cost nothing.
      const unenrichedRealtime = realtimeLogs.filter((log) => !log.isEnriched);
      const logsToEnrich = [...newLogs, ...unenrichedRealtime];
      if (logsToEnrich.length > 0) {
        fetchEnrichment(logsToEnrich);
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

  // Set up polling effect — only poll on page 1
  useEffect(() => {
    if (startPolling && queryPage === 1) {
      const interval = setInterval(pollForNewLogs, pollIntervalMs);
      return () => clearInterval(interval);
    }
  }, [startPolling, queryPage, pollForNewLogs, pollIntervalMs]);

  // Reset realtime logs when polling stops
  useEffect(() => {
    if (!startPolling) {
      setRealtimeLogsMap(new Map());
    }
  }, [startPolling]);

  // When live tail turns on, snap the URL back to page 1. queryPage already
  // forces 1 synchronously; this commits it to URL state on the rising edge of
  // startPolling so the page state reflects what the user is actually viewing.
  const prevStartPollingRef = useRef(startPolling);
  useEffect(() => {
    const justEnabled = startPolling && !prevStartPollingRef.current;
    prevStartPollingRef.current = startPolling;
    if (justEnabled) {
      setPage(1);
    }
  }, [startPolling, setPage]);

  const isInitialLoading = isLoading && !logData;
  const isNavigating = isFetching && !isInitialLoading;

  return {
    realtimeLogs,
    historicalLogs,
    totalCount,
    isLoading: isInitialLoading,
    isFetching,
    isNavigating,
    isPolling: startPolling,
    page: queryPage,
    pageSize: limit,
    totalPages,
    onPageChange,
    sorting,
    onSortingChange: setSorting,
  };
}

const sortLogs = (logs: EnrichedRatelimitLog[]) => {
  return logs.toSorted((a, b) => b.time - a.time);
};
