import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshButton } from "@unkey/ui";
import { useGatewayLogsContext } from "../../../context/gateway-logs-provider";

export const GatewayLogsRefresh = () => {
  const { toggleLive, isLive } = useGatewayLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = async () => {
    refreshQueryTime();
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: trpc.logs.queryLogs.queryKey(),
      }),
      queryClient.invalidateQueries({
        queryKey: trpc.logs.queryTimeseries.queryKey(),
      }),
    ]);
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
