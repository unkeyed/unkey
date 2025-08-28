import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useRouter } from "next/navigation";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();
  const { ratelimit } = trpc.useUtils();
  const { refresh } = useRouter();

  const handleRefresh = () => {
    refreshQueryTime();
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
    refresh();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
