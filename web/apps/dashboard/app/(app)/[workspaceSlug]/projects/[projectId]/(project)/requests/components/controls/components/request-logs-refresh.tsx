import { RefreshButton } from "@unkey/ui";
import { trpc } from "@/lib/trpc/client";
import { useRequestLogsContext } from "../../../context/request-logs-provider";

export const RequestLogsRefresh = () => {
  const { deploy } = trpc.useUtils();
  const { refresh } = useRequestLogsContext();

  const handleRefresh = () => {
    // Re-anchor the query window so newly arrived logs are included, then drop
    // cached pages so they refetch against the new window.
    refresh();
    deploy.requestLogs.query.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
