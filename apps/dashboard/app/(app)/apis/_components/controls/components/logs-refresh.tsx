import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { api } = trpc.useUtils();

  const handleRefresh = () => {
    api.overview.timeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
