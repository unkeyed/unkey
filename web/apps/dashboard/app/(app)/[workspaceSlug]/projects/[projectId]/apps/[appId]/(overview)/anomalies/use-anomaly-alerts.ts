import { trpc } from "@/lib/trpc/client";

type UseAnomalyAlertsParams = {
  appId: string;
  environmentId: string;
  startMs: number;
  endMs: number;
  enabled: boolean;
};

export function useAnomalyAlerts({ enabled, ...range }: UseAnomalyAlertsParams) {
  const query = trpc.alerts.list.useInfiniteQuery(
    { ...range, includeResolved: true, limit: 25 },
    {
      enabled,
      staleTime: 30_000,
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  return {
    ...query,
    alerts: query.data?.pages.flatMap((page) => page.alerts) ?? [],
  };
}
