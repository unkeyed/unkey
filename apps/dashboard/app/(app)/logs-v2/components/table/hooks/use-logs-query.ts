import { trpc } from "@/lib/trpc/client";
import type { Log } from "@unkey/clickhouse/src/logs";
import { useEffect, useMemo, useState } from "react";
import type { z } from "zod";
import { useFilters } from "../../../hooks/use-filters";
import type { queryLogsPayload } from "../query-logs.schema";

interface UseLogsQueryParams {
  limit?: number;
}

export function useLogsQuery({ limit = 50 }: UseLogsQueryParams = {}) {
  const [logs, setLogs] = useState<Log[]>([]);
  const { filters } = useFilters();

  // Without this initial request happens twice
  const timestamps = useMemo(
    () => ({
      startTime: Date.now() - 24 * 60 * 60 * 1000,
      endTime: Date.now(),
    }),
    [],
  );

  const queryParams = useMemo(() => {
    const params: z.infer<typeof queryLogsPayload> = {
      limit,
      startTime: timestamps.startTime,
      endTime: timestamps.endTime,
      host: { filters: [] },
      requestId: { filters: [] },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
    };

    // Process each filter
    filters.forEach((filter) => {
      switch (filter.field) {
        case "startTime":
        case "endTime":
          params[filter.field] = filter.value as number;
          break;

        case "status": {
          params.status?.filters.push({
            operator: "is",
            value: Number.parseInt(filter.value as string),
          });
          break;
        }

        case "methods": {
          if (typeof filter.value !== "string") {
            console.error("Method filter value type has to be 'string'");
          }

          params.method?.filters.push({
            operator: "is",
            value: filter.value as string,
          });
          break;
        }

        case "paths": {
          if (typeof filter.value !== "string") {
            console.error("Path filter value type has to be 'string'");
          }

          params.path?.filters.push({
            operator: filter.operator,
            value: filter.value as string,
          });
          break;
        }

        case "host": {
          if (typeof filter.value !== "string") {
            console.error("Host filter value type has to be 'string'");
          }

          params.host?.filters.push({
            operator: "is",
            value: filter.value as string,
          });
          break;
        }

        case "requestId": {
          if (typeof filter.value !== "string") {
            console.error("Request ID filter value type has to be 'string'");
          }

          params.requestId?.filters.push({
            operator: "is",
            value: filter.value as string,
          });
          break;
        }
      }
    });

    return params;
  }, [filters, limit, timestamps]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.logs.queryLogs.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialCursor: { requestId: null, time: null },
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // // Query for new logs (polling)
  // const pollForNewLogs = async () => {
  //   if (!isLive || !logs[0]) return;
  //
  //   const pollParams = {
  //     ...queryParams,
  //     limit: 10,
  //     startTime: logs[0].time,
  //     endTime: Date.now(),
  //   };
  //
  //   const result = await queryClient.fetchQuery({
  //     queryKey: trpc.logs.queryLogs.getQueryKey(pollParams),
  //     queryFn: () => trpc.logs.queryLogs.fetch(pollParams),
  //   });
  //
  //   if (result.logs.length > 0) {
  //     const newLogs = result.logs.filter(
  //       (newLog) =>
  //         !logs.some(
  //           (existingLog) => existingLog.request_id === newLog.request_id
  //         )
  //     );
  //     if (newLogs.length > 0) {
  //       setLogs((prev) => [...newLogs, ...prev]);
  //     }
  //   }
  // };
  //
  useEffect(() => {
    if (initialData) {
      const allLogs = initialData.pages.flatMap((page) => page.logs);
      setLogs(allLogs);
    }
  }, [initialData]);

  // // Set up polling
  // useEffect(() => {
  //   if (isLive) {
  //     pollInterval.current = window.setInterval(pollForNewLogs, 5000);
  //   }
  //
  //   return () => {
  //     if (pollInterval.current) {
  //       clearInterval(pollInterval.current);
  //     }
  //   };
  // }, [isLive, logs[0]?.time, queryParams]);
  //
  // const toggleLive = () => {
  //   setIsLive((prev) => !prev);
  //   if (!isLive) {
  //     refetch();
  //   }
  // };

  return {
    logs,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
