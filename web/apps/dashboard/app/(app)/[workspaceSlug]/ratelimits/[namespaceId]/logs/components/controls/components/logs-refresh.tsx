import { useAnchoredRefresh } from "@/hooks/use-anchored-refresh";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import { RefreshButton } from "@unkey/ui";
import { useRatelimitLogsContext } from "../../../context/logs";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useRatelimitLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const handleRefresh = useAnchoredRefresh(refreshQueryTime, [
    getQueryKey(trpc.ratelimit.logs.query),
    getQueryKey(trpc.ratelimit.logs.queryRatelimitTimeseries),
  ]);

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
