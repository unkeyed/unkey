"use client";

import { useApp, useAppScope, warmDeploymentsPage } from "@/lib/deploy-data";
import { PageContainer } from "@unkey/ui";
import { useEffect } from "react";
import { ActiveBranches } from "./components/active-branches";
import { MetricsGlance } from "./components/metrics-glance";
import { ProductionOverview } from "./components/production-overview";
import { RecentDeployments } from "./components/recent-deployments";

// Most visits to an app move on to its deployments, so the overview loads that
// list once its own reads have settled
const DEPLOYMENTS_WARM_DELAY_MS = 1000;

export function Overview() {
  const { data: app } = useApp();
  const { projectId, appId } = useAppScope();
  useEffect(() => {
    const timer = setTimeout(
      () => warmDeploymentsPage(projectId, appId),
      DEPLOYMENTS_WARM_DELAY_MS,
    );
    return () => clearTimeout(timer);
  }, [projectId, appId]);

  return (
    <PageContainer>
      <ProductionOverview>
        <MetricsGlance />
        {app?.sourceType === "oci" ? <RecentDeployments /> : <ActiveBranches />}
      </ProductionOverview>
    </PageContainer>
  );
}
