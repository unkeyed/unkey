"use client";

import { PageContainer } from "@unkey/ui";
import { useAppCurrentDeployment } from "../hooks/use-app-current-deployment";
import { ActiveBranches } from "./components/active-branches";
import { AppProductionCard } from "./components/app-production-card";
import { MetricsGlance } from "./components/metrics-glance";
import { RecentDeployments } from "./components/recent-deployments";

export function Overview() {
  const { app } = useAppCurrentDeployment();

  return (
    <PageContainer>
      <AppProductionCard>
        <MetricsGlance />
        {app?.sourceType === "oci" ? <RecentDeployments /> : <ActiveBranches />}
      </AppProductionCard>
    </PageContainer>
  );
}
