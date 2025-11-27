import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useRatelimitLogsContext } from "../../../context/logs";
import { useQueryClient } from "@tanstack/react-query";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useRatelimitLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = () => {
    refreshQueryTime();
    queryClient.invalidateQueries({ queryKey: trpc.ratelimit.logs.query.queryKey() });
    queryClient.invalidateQueries({ queryKey: trpc.ratelimit.logs.queryRatelimitTimeseries.queryKey() });
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
