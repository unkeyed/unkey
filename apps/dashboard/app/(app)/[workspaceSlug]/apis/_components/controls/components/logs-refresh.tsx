import { useTRPC } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";
import { useQueryClient } from "@tanstack/react-query";

export const LogsRefresh = () => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = () => {
    queryClient.invalidateQueries({
      queryKey: trpc.api.overview.timeseries.queryKey(),
    });
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
