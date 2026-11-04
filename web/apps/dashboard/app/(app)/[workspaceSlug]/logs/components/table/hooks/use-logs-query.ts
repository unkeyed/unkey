import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useLiveTail } from "@/hooks/use-live-tail";
import { listApiRequests, toApiRequestsFilters } from "@/lib/api-request-logs";
import { shownTotalOf, tableStateOf, useInfiniteList } from "@/lib/loadable/list";
import { queryKeys } from "@/lib/query-keys";
import { useQueryTime } from "@/providers/query-time-provider";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useFilters } from "../../../hooks/use-filters";
import { useRequestTotal } from "../../charts/hooks/use-fetch-timeseries";

const PAGE_SIZE = 50;
const LIVE_POLL_INTERVAL_MS = 2_000;

type ApiRequestsPage = Awaited<ReturnType<typeof listApiRequests>>;

const logsOf = (page: ApiRequestsPage) => page.logs;
const requestIdOf = (log: ApiRequestsPage["logs"][number]) => log.request_id;

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

  const table = tableStateOf(useInfiniteList(pages, logsOf));
  const historicalLogs = table.rows;
  const requestTotal = useRequestTotal();

  const realtimeLogs = useLiveTail({
    enabled: isLive,
    intervalMs: LIVE_POLL_INTERVAL_MS,
    queryKey: queryKeys.logs.apiRequests.live(apiFilters, PAGE_SIZE),
    windowStart: apiFilters.time?.gte ?? timestamp - HISTORICAL_DATA_WINDOW,
    fetchNewest: async (window, signal) => {
      const page = await listApiRequests(
        {
          limit: PAGE_SIZE,
          filters: { ...apiFilters, time: { gte: window.startTime, lte: window.endTime } },
        },
        signal,
      );
      return page.logs;
    },
    keyOf: requestIdOf,
    historical: historicalLogs,
  });

  return {
    realtimeLogs,
    table,
    totalCount: shownTotalOf({
      loaded: historicalLogs.length,
      total: requestTotal,
      hasMore: pages.hasNextPage === true,
    }),
  };
}
