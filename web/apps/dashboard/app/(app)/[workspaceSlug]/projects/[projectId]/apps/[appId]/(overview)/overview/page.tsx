"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { ActiveDeploymentCardEmpty } from "../../components/active-deployment-card/components/active-deployment-card-empty";
import { useProjectData } from "../data-provider";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { ActiveBranches } from "./components/active-branches";
import { AppProductionCard } from "./components/app-production-card";
import { OverviewPageTitle } from "./components/overview-page-title";

export default function Overview() {
  const { deployments, isDeploymentsLoading } = useProjectData();
  const hasNoDeployments = !isDeploymentsLoading && deployments.length === 0;

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
        {hasNoDeployments ? (
          <CreateDeploymentButton
            renderTrigger={({ onClick }) => (
              <ActiveDeploymentCardEmpty onCreateDeployment={onClick} />
            )}
          />
        ) : (
          <div className="flex flex-col gap-5">
            <AppProductionCard />
            <ActiveBranches />
          </div>
        )}
      </PageBody>
    </PageContainer>
  );
}
