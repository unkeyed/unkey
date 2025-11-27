import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = async () => {
    refreshQueryTime();
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: trpc.ratelimit.overview.logs.query.queryKey(),
      }),
      queryClient.invalidateQueries({
        queryKey: trpc.ratelimit.overview.logs.queryRatelimitLatencyTimeseries.queryKey(),
      }),
      queryClient.invalidateQueries({
        queryKey: trpc.ratelimit.logs.queryRatelimitTimeseries.queryKey(),
      }),
    ]);
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
