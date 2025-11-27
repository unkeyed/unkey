import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useQueryClient } from "@tanstack/react-query";
import { useLogsContext } from "../../../context/logs";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = async () => {
    refreshQueryTime();
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: trpc.logs.queryLogs.queryKey()
      }),
      queryClient.invalidateQueries({
        queryKey: trpc.logs.queryTimeseries.queryKey()
      })
    ]);
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
