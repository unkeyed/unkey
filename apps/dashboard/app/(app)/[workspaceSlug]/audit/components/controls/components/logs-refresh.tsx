import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { audit } = trpc.useUtils();
  const { refreshQueryTime } = useQueryTime();

  const handleRefresh = () => {
    refreshQueryTime();
    audit.logs.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
