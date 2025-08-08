import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const { filters } = useFilters();
  const { logs } = trpc.useUtils();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    refreshQueryTime();
    logs.queryLogs.invalidate();
    logs.queryTimeseries.invalidate();
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
