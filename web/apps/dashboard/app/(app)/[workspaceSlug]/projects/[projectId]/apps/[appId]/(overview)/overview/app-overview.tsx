"use client";

import { PageContainer } from "@unkey/ui";
import { useAppCurrentDeployment } from "../hooks/use-app-current-deployment";
import { ActiveBranches } from "./components/active-branches";
import { MetricsGlance } from "./components/metrics-glance";
import { ProductionOverview } from "./components/production-overview";
import { RecentDeployments } from "./components/recent-deployments";

export function Overview() {
  const { app } = useAppCurrentDeployment();

  return (
    <PageContainer>
      <ProductionOverview>
        <MetricsGlance />
        {app?.sourceType === "oci" ? <RecentDeployments /> : <ActiveBranches />}
      </ProductionOverview>
    </PageContainer>
  );
}
