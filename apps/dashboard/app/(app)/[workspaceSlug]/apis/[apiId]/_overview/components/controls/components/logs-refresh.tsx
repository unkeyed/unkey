import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useQueryClient } from "@tanstack/react-query";
import { useTRPC } from "@/lib/trpc/client";

export const LogsRefresh = () => {
  const trpc = useTRPC()
  const { refreshQueryTime } = useQueryTime();
  const queryClient = useQueryClient();

  const handleRefresh = () => {
    refreshQueryTime();
    queryClient.invalidateQueries({ queryKey: trpc.api.keys.query.queryKey() });
    queryClient.invalidateQueries({ queryKey: trpc.api.keys.timeseries.queryKey() });
    queryClient.invalidateQueries({ queryKey: trpc.api.keys.activeKeysTimeseries.queryKey() });
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
