"use client";

import { RuntimeLogsView } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/runtime-logs-view";

// Scoped to this deployment: useRuntimeLogsQuery reads the deploymentId route
// param, and the deployment filter is hidden on deployment routes.
export default function DeploymentLogsPage() {
  return <RuntimeLogsView />;
}
