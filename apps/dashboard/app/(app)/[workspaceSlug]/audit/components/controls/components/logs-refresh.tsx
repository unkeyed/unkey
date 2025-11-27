import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useQueryClient } from "@tanstack/react-query";

export const LogsRefresh = () => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const { refreshQueryTime } = useQueryTime();

  const handleRefresh = () => {
    refreshQueryTime();
    queryClient.invalidateQueries({
      queryKey: trpc.audit.logs.queryKey()
    });
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
