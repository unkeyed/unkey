import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";

import { useQuery } from "@tanstack/react-query";

type useFetchRequestDetails = {
  requestId?: string;
};

export function useFetchRequestDetails({ requestId }: useFetchRequestDetails) {
  const trpc = useTRPC();
  const { queryTime: timestamp } = useQueryTime();
  const query = useQuery(
    trpc.logs.queryLogs.queryOptions(
      {
        limit: 1,
        startTime: 0,
        endTime: timestamp,
        host: { filters: [] },
        method: { filters: [] },
        path: { filters: [] },
        status: { filters: [] },
        requestId: requestId
          ? {
              filters: [
                {
                  operator: "is",
                  value: requestId,
                },
              ],
            }
          : null,
        since: "",
      },
      {
        enabled: Boolean(requestId),
        refetchOnWindowFocus: false,
        refetchOnMount: false,
      },
    ),
  );

  return {
    log: query.data?.logs[0],
    isLoading: query.isLoading,
    error: query.error,
  };
}
