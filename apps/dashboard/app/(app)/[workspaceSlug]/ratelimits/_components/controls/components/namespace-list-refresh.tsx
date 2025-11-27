import { useTRPC } from "@/lib/trpc/client";
import { useQueryClient } from "@tanstack/react-query";
import { RefreshButton } from "@unkey/ui";

export const NamespaceListRefresh = () => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const handleRefresh = () => {
    queryClient.invalidateQueries(trpc.ratelimit.logs.queryRatelimitTimeseries.pathFilter());
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
