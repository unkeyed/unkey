import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useQueryClient } from "@tanstack/react-query";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = async () => {
    refreshQueryTime();
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: trpc.api.keys.query.queryKey(),
      }),
      queryClient.invalidateQueries({
        queryKey: trpc.api.keys.timeseries.queryKey(),
      }),
      queryClient.invalidateQueries({
        queryKey: trpc.api.keys.activeKeysTimeseries.queryKey(),
      }),
    ]);
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
