"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { ActiveDeploymentCardEmpty } from "../../components/active-deployment-card/components/active-deployment-card-empty";
import { useProjectData } from "../data-provider";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { OverviewPageTitle } from "./components/overview-page-title";
import { ProductionDeploymentCard } from "./components/production-deployment-card";
import { RecentDeployments } from "./components/recent-deployments";

export default function Overview() {
  const { deployments, isDeploymentsLoading } = useProjectData();
  const hasNoDeployments = !isDeploymentsLoading && deployments.length === 0;

  return (
    // 52px is TOP_NAV_HEIGHT.
    <PageContainer className={hasNoDeployments ? "min-h-[calc(100dvh-52px)]" : undefined}>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>
            <OverviewPageTitle />
          </PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      {hasNoDeployments ? (
        <PageBody className="flex-1">
          <CreateDeploymentButton
            renderTrigger={({ onClick }) => (
              <ActiveDeploymentCardEmpty onCreateDeployment={onClick} className="h-full flex-1" />
            )}
          />
        </PageBody>
      ) : (
        <PageBody>
          <div className="flex flex-col gap-5">
            <ProductionDeploymentCard />
            <RecentDeployments />
          </div>
        </PageBody>
      )}
    </PageContainer>
  );
}
