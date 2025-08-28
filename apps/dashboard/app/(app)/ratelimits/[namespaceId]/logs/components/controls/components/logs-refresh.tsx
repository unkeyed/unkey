import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useRatelimitLogsContext } from "../../../context/logs";

export const LogsRefresh = () => {
  const { toggleLive, isLive } = useRatelimitLogsContext();
  const { refreshQueryTime } = useQueryTime();
  const { ratelimit } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    ratelimit.logs.query.invalidate();
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
};
