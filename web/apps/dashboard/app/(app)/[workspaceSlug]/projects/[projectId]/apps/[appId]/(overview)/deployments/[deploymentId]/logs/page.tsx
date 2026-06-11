"use client";

import { PageContainer } from "@unkey/ui";

// TODO(deployment-logs): replace with the runtime logs view pinned to this
// deployment. Extract a shared RuntimeLogsView from the project logs route
// that accepts a fixed deploymentId and hides the deployment filter.
export default function DeploymentLogsPage() {
  return (
    <PageContainer width="full" className="px-6 py-6">
      <p className="text-sm text-gray-11">Deployment logs are coming here next.</p>
    </PageContainer>
  );
}
