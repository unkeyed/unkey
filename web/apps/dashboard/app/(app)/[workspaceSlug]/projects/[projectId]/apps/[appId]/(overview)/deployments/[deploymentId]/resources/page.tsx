"use client";

import { PageContainer } from "@unkey/ui";
import { DeploymentResources } from "./deployment-resources";

export default function DeploymentResourcesPage() {
  return (
    <PageContainer className="mx-auto max-w-7xl px-6 py-6">
      <DeploymentResources />
    </PageContainer>
  );
}
