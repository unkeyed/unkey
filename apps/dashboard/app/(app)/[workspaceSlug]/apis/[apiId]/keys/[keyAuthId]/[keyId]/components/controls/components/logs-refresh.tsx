import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const { api } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    api.keys.query.invalidate();
    api.keys.timeseries.invalidate();
    api.keys.activeKeysTimeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
