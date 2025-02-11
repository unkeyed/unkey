import { RefreshButton } from "@/components/logs/refresh-button";
import { useTRPC } from "@/lib/trpc/client";
import { useRatelimitLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";
import { useQueryClient } from "@tanstack/react-query";
export const LogsRefresh = () => {
  const { toggleLive, isLive } = useRatelimitLogsContext();
  const { filters } = useFilters();
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    queryClient.invalidateQueries(trpc.ratelimit.logs.query.queryFilter());
    queryClient.invalidateQueries(trpc.ratelimit.logs.queryRatelimitTimeseries.queryFilter());
  };

  return (
    <RefreshButton
      onRefresh={handleRefresh}
      isEnabled={Boolean(hasRelativeFilter)}
      isLive={isLive}
      toggleLive={toggleLive}
    />
  );
};
