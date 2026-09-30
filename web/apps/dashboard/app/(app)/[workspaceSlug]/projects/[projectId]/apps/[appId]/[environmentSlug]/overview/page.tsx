"use client";

import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import type { ComponentType } from "react";
import { ActiveDeploymentCardEmpty } from "../../components/active-deployment-card/components/active-deployment-card-empty";
import { useProjectData } from "../../data-provider";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { useAppEnvironment } from "../environment-context";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { ActiveBranches } from "./components/active-branches";
import { AppProductionCard } from "./components/app-production-card";
import { LatestDeployments } from "./components/latest-deployments";
import { OverviewPageTitle } from "./components/overview-page-title";

const historySectionByKind: Record<EnvironmentKind, ComponentType> = {
  production: LatestDeployments,
  preview: ActiveBranches,
};

export default function Overview() {
  const { deployments, isDeploymentsLoading } = useProjectData();
  const { app } = useAppCurrentDeployment();
  const { environment } = useAppEnvironment();
  const HistorySection =
    app?.sourceType === "oci" ? LatestDeployments : historySectionByKind[environment.kind];
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
            <HistorySection />
          </div>
        )}
      </PageBody>
    </PageContainer>
  );
}
