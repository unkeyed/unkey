import { useAnchoredRefresh } from "@/hooks/use-anchored-refresh";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const handleRefresh = useAnchoredRefresh(refreshQueryTime, [
    getQueryKey(trpc.ratelimit.overview.logs.query),
    getQueryKey(trpc.ratelimit.overview.logs.queryRatelimitLatencyTimeseries),
    getQueryKey(trpc.ratelimit.logs.queryRatelimitTimeseries),
  ]);

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
