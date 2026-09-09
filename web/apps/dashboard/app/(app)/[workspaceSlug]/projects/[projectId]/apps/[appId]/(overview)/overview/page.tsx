"use client";

import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { ActiveDeploymentCardEmpty } from "../../components/active-deployment-card/components/active-deployment-card-empty";
import { useProjectData } from "../data-provider";
import { useAppCurrentDeployment } from "../hooks/use-app-current-deployment";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { ActiveBranches } from "./components/active-branches";
import { AppProductionCard } from "./components/app-production-card";
import { OverviewPageTitle } from "./components/overview-page-title";
import { RecentDeployments } from "./components/recent-deployments";

export default function Overview() {
  const { deployments, isDeploymentsLoading } = useProjectData();
  const { app } = useAppCurrentDeployment();
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
            {app?.sourceType === "oci" ? <RecentDeployments /> : <ActiveBranches />}
          </div>
        )}
      </PageBody>
    </PageContainer>
  );
}
