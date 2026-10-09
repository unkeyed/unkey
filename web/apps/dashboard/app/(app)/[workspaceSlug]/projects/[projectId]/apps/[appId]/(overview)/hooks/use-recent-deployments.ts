"use client";

import { useLiveQuery } from "@tanstack/react-db";
import { useProjectData } from "../data-provider";
import { recentDeploymentsQueryFor } from "../data-provider-queries";

// The newest deployments of the app, or of the whole project outside an app.
// Pages and dialogs subscribe when they need them and share one load
export function useRecentDeployments({ enabled = true }: { enabled?: boolean } = {}) {
  const { projectId, appId } = useProjectData();
  const query = useLiveQuery(
    (q) => (enabled ? recentDeploymentsQueryFor(projectId, appId)(q) : null),
    [projectId, appId, enabled],
  );
  return { deployments: query.data ?? [], isLoading: enabled && query.isLoading };
}
