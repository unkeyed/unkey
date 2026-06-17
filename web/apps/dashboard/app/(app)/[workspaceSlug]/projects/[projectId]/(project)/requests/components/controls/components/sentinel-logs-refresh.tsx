import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";
import { useSentinelLogsContext } from "../../../context/sentinel-logs-provider";

export const SentinelLogsRefresh = () => {
  const { deploy } = trpc.useUtils();
  const { refresh } = useSentinelLogsContext();

  const handleRefresh = () => {
    // Re-anchor the query window so newly arrived logs are included, then drop
    // cached pages so they refetch against the new window.
    refresh();
    deploy.sentinelLogs.query.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
