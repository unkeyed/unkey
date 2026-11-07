import { useAnchoredRefresh } from "@/hooks/use-anchored-refresh";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const handleRefresh = useAnchoredRefresh(refreshQueryTime, [
    getQueryKey(trpc.identity.logs.query),
    getQueryKey(trpc.identity.logs.timeseries),
  ]);

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
