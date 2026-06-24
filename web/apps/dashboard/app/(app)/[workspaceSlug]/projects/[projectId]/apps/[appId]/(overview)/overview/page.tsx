"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
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
        <div className="flex flex-col gap-5">
          <ProductionDeploymentCard />
          <RecentDeployments />
        </div>
      </PageBody>
    </PageContainer>
  );
}
