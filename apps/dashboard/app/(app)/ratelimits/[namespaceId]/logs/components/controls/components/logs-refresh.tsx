import { RefreshButton } from "@/components/logs/refresh-button";
import { trpc } from "@/lib/trpc/client";
import { useRatelimitLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";
import { refreshQueryTimestamp } from "../../table/hooks/utils";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useRatelimitLogsContext();
  const { filters } = useFilters();
  const { ratelimit } = trpc.useUtils();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    refreshQueryTimestamp();
    ratelimit.logs.query.invalidate();
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
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
