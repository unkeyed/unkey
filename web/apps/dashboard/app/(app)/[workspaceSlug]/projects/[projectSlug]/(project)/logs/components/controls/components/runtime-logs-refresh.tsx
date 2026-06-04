"use client";

import { useRuntimeLogs } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/logs/context/runtime-logs-provider";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

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
