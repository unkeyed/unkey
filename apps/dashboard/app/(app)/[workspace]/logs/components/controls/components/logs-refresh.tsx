import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useLogsContext } from "../../../context/logs";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useLogsContext();
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
