"use client";

import { SentinelLogsView } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/sentinel-logs-view";
import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { useDeploymentNavVariant } from "../use-deployment-nav-variant";

// Scoped to this deployment: useSentinelLogsQuery reads the deploymentId route
// param, and the deployment filter is hidden on deployment routes. In the
// sidebar variant there are no tabs, so the page carries its own header.
export default function DeploymentRequestsPage() {
  const [navVariant] = useDeploymentNavVariant();

  return (
    <PageContainer width="full">
      {navVariant === "sidebar" && (
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Requests</PageHeaderTitle>
          </PageHeaderContent>
        </PageHeader>
      )}
      <SentinelLogsView />
    </PageContainer>
  );
}
