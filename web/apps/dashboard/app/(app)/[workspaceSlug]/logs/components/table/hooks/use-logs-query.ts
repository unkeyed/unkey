import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { listApiRequests, toApiRequestsFilters } from "@/lib/api-request-logs";
import { queryKeys } from "@/lib/query-keys";
import { useQueryTime } from "@/providers/query-time-provider";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useFilters } from "../../../hooks/use-filters";

const PAGE_SIZE = 50;
const LIVE_POLL_INTERVAL_MS = 2_000;

export function useLogsQuery({ isLive }: { isLive: boolean }) {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const apiFilters = useMemo(
    () =>
      toApiRequestsFilters(filters, {
        startTime: timestamp - HISTORICAL_DATA_WINDOW,
        endTime: timestamp,
      }),
    [filters, timestamp],
  );

  const pages = useInfiniteQuery({
    queryKey: queryKeys.logs.apiRequests.list(apiFilters, PAGE_SIZE),
    queryFn: ({ pageParam, signal }) =>
      listApiRequests(
        {
          limit: PAGE_SIZE,
          cursor: typeof pageParam === "string" ? pageParam : undefined,
          filters: apiFilters,
        },
        signal,
      ),
    getNextPageParam: (lastPage) => lastPage.cursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Live mode re-reads the newest rows with no upper time bound. API instances
  // flush their logs late and out of order, so asking for the newest rows each
  // time, instead of the rows after the last one seen, also catches late rows
  const liveFilters = { ...apiFilters, time: { gte: apiFilters.time?.gte } };
  const live = useQuery({
    queryKey: queryKeys.logs.apiRequests.live(liveFilters, PAGE_SIZE),
    queryFn: ({ signal }) => listApiRequests({ limit: PAGE_SIZE, filters: liveFilters }, signal),
    enabled: isLive,
    refetchInterval: LIVE_POLL_INTERVAL_MS,
  });

  const historicalLogs = useMemo(
    () => pages.data?.pages.flatMap((page) => page.logs) ?? [],
    [pages.data],
  );

  const realtimeLogs = useMemo(() => {
    if (!isLive) {
      return [];
    }
    const historicalIds = new Set(historicalLogs.map((log) => log.request_id));
    return (live.data?.logs ?? []).filter((log) => !historicalIds.has(log.request_id));
  }, [isLive, live.data, historicalLogs]);

  return {
    realtimeLogs,
    historicalLogs,
    isLoading: pages.isLoading,
    hasMore: pages.hasNextPage,
    loadMore: pages.fetchNextPage,
    isLoadingMore: pages.isFetchingNextPage,
  };
}
