"use client";

import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";
import { useRuntimeLogs } from "../../../context/runtime-logs-provider";

export function RuntimeLogsRefresh() {
  const { isLive, toggleLive } = useRuntimeLogs();
  const { refreshQueryTime } = useQueryTime();
  const {
    deploy: { runtimeLogs },
  } = trpc.useUtils();

  const handleRefresh = () => {
    refreshQueryTime();
    runtimeLogs.query.invalidate();
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
}
