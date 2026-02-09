import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";

export const SentinelLogsRefresh = () => {
  const { deploy } = trpc.useUtils();

  const handleRefresh = () => {
    deploy.sentinelLogs.query.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
