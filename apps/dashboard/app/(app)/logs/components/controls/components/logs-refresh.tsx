import { RefreshButton } from "@/components/logs/refresh-button";
import { useTRPC } from "@/lib/trpc/client";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";
import { useQueryClient } from "@tanstack/react-query";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useLogsContext();
  const { filters } = useFilters();
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    queryClient.invalidateQueries(trpc.logs.queryLogs.queryFilter());
    queryClient.invalidateQueries(trpc.logs.queryTimeseries.queryFilter());
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
