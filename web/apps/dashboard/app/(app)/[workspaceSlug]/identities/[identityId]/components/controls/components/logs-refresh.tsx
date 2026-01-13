import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const { identity } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    identity.logs.query.invalidate();
    identity.logs.timeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};