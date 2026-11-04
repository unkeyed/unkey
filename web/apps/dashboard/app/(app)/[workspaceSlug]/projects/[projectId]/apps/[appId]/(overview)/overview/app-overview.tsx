"use client";

import { useApp } from "@/lib/deploy-data";
import { PageContainer } from "@unkey/ui";
import { ActiveBranches } from "./components/active-branches";
import { MetricsGlance } from "./components/metrics-glance";
import { ProductionOverview } from "./components/production-overview";
import { RecentDeployments } from "./components/recent-deployments";

export function Overview() {
  const { data: app } = useApp();

  return (
    <PageContainer>
      <ProductionOverview>
        <MetricsGlance />
        {app?.sourceType === "oci" ? <RecentDeployments /> : <ActiveBranches />}
      </ProductionOverview>
    </PageContainer>
  );
}
