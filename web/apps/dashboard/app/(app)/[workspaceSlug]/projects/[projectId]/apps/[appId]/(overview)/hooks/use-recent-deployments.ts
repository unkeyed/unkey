"use client";

import { DEPLOYMENT_PICKER_LIMIT, useRecentDeployments as useRecent } from "@/lib/deploy-data";

// Legacy pages only. Everything else reads through @/lib/deploy-data.
export function useRecentDeployments({ enabled = true }: { enabled?: boolean } = {}) {
  const { data, isLoading } = useRecent({ limit: DEPLOYMENT_PICKER_LIMIT, enabled });
  return { deployments: data, isLoading };
}
