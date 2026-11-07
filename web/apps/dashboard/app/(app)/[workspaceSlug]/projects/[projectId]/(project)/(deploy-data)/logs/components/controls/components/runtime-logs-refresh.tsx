"use client";

import { useRuntimeLogs } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/(deploy-data)/logs/context/runtime-logs-provider";
import { useAnchoredRefresh } from "@/hooks/use-anchored-refresh";
import { trpc } from "@/lib/trpc/client";
import { getQueryKey } from "@trpc/react-query";
import { RefreshButton } from "@unkey/ui";

export function RuntimeLogsRefresh() {
  const { isLive, toggleLive, refresh } = useRuntimeLogs();
  const handleRefresh = useAnchoredRefresh(refresh, [getQueryKey(trpc.deploy.runtimeLogs.query)]);

  return (
    <RefreshButton onRefresh={handleRefresh} isEnabled isLive={isLive} toggleLive={toggleLive} />
  );
}
