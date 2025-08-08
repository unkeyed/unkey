import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useFilters } from "../../../hooks/use-filters";

export const LogsRefresh = () => {
  const { filters } = useFilters();
  const { refreshQueryTime } = useQueryTime();
  const { api } = trpc.useUtils();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    refreshQueryTime();
    api.keys.query.invalidate();
    api.keys.timeseries.invalidate();
    api.keys.activeKeysTimeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled={Boolean(hasRelativeFilter)} />;
};
