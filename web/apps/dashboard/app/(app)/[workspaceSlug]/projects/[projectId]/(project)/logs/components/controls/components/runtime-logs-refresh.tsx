"use client";

import { useRuntimeLogs } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/context/runtime-logs-provider";
import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";

export function RuntimeLogsRefresh() {
  const { isLive, toggleLive, refresh } = useRuntimeLogs();
  const {
    deploy: { runtimeLogs },
  } = trpc.useUtils();

  const handleRefresh = () => {
    // Re-anchor the query window so newly arrived logs are included, then drop
    // cached pages so they refetch against the new window.
    refresh();
    runtimeLogs.query.invalidate();
  };

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
}
