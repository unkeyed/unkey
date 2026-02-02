"use client";

import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

export function RuntimeLogsRefresh() {
  // const { toggleLive, isLive } = useRuntimeLogs();
  const { refreshQueryTime } = useQueryTime();
  const { deploy: { runtimeLogs
  } } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    runtimeLogs.query.invalidate();
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled />
  );
}
