import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const { ratelimit } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    ratelimit.overview.logs.query.invalidate();
    ratelimit.overview.logs.queryRatelimitLatencyTimeseries.invalidate();
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
