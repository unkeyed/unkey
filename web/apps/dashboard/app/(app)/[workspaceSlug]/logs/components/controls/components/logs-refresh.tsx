import { queryKeys } from "@/lib/query-keys";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshButton } from "@unkey/ui";
import { useLogsContext } from "../../../context/logs";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const { logs } = trpc.useUtils();
  const queryClient = useQueryClient();

  const handleRefresh = () => {
    refreshQueryTime();
    queryClient.invalidateQueries({ queryKey: queryKeys.logs.apiRequests.all });
    logs.queryTimeseries.invalidate();
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
