import { RefreshButton } from "@/components/logs/refresh-button";
import { trpc } from "@/lib/trpc/client";
import { useFilters } from "../../hooks/use-filters";

export const LogsRefresh = () => {
  const { filters } = useFilters();
  const { ratelimit } = trpc.useUtils();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    ratelimit.overview.logs.queryRatelimitLatencyTimeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled={Boolean(hasRelativeFilter)} />;
};
