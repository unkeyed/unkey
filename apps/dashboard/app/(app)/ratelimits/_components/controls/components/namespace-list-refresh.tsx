import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";

export const NamespaceListRefresh = () => {
  const { ratelimit } = trpc.useUtils();

  const handleRefresh = () => {
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
