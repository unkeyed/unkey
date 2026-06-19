"use client";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { OverviewDebugNav } from "./components/overview-debug-nav";
import { OverviewPageTitle } from "./components/overview-page-title";
import { ProductionDeploymentCard } from "./components/production-deployment-card";
import { RecentDeployments } from "./components/recent-deployments";

export default function Overview() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>
            <OverviewPageTitle />
          </PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <ProductionDeploymentCard />
        <RecentDeployments />
      </PageBody>
      <OverviewDebugNav />
    </PageContainer>
  );
}
