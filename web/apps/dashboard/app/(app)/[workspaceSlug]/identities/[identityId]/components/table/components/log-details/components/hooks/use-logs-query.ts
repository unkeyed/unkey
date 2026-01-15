import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";

type useFetchRequestDetails = {
  requestId?: string;
};

export function useFetchRequestDetails({ requestId }: useFetchRequestDetails) {
  const { queryTime: timestamp } = useQueryTime();
  const query = trpc.logs.queryLogs.useQuery(
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
  );

  return {
    log: query.data?.logs[0],
    isLoading: query.isLoading,
    error: query.error,
  };
}
