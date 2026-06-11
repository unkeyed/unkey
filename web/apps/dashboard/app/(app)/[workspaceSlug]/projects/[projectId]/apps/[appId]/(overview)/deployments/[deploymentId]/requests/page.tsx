"use client";

import { SentinelLogsView } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/sentinel-logs-view";

// Scoped to this deployment: useSentinelLogsQuery reads the deploymentId route
// param, and the deployment filter is hidden on deployment routes.
export default function DeploymentRequestsPage() {
  return <SentinelLogsView />;
}
