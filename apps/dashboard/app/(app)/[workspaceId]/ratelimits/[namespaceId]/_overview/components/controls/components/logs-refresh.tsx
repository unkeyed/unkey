import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useFilters } from "../../../hooks/use-filters";

export const LogsRefresh = () => {
  const { filters } = useFilters();
  const { refreshQueryTime } = useQueryTime();

  const { ratelimit } = trpc.useUtils();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    refreshQueryTime();
    ratelimit.overview.logs.query.invalidate();
    ratelimit.overview.logs.queryRatelimitLatencyTimeseries.invalidate();
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled={Boolean(hasRelativeFilter)} />;
};
