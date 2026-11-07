import { useAnchoredRefresh } from "@/hooks/use-anchored-refresh";
import { trpc } from "@/lib/trpc/client";
import { getQueryKey } from "@trpc/react-query";
import { RefreshButton } from "@unkey/ui";
import { useRequestLogsContext } from "../../../context/request-logs-provider";

export const RequestLogsRefresh = () => {
  const { refresh } = useRequestLogsContext();
  const handleRefresh = useAnchoredRefresh(refresh, [getQueryKey(trpc.deploy.requestLogs.query)]);

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
