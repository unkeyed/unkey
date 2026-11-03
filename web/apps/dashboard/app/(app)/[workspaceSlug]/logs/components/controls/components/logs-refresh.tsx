import { useAnchoredRefresh } from "@/hooks/use-anchored-refresh";
import { queryKeys } from "@/lib/query-keys";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useLogsContext } from "../../../context/logs";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const handleRefresh = useAnchoredRefresh(refreshQueryTime, [queryKeys.logs.apiRequests.all]);

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
