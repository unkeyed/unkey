import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useSentinelLogsContext } from "../../../context/sentinel-logs-provider";

export const SentinelLogsRefresh = () => {
  const { toggleLive, isLive } = useSentinelLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const { logs } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    logs.queryLogs.invalidate();
    logs.queryTimeseries.invalidate();
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
