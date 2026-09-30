"use client";

import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import { IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, PageBody, PageContainer } from "@unkey/ui";
import type { ComponentType } from "react";
import { ActiveDeploymentCardEmpty } from "../../components/active-deployment-card/components/active-deployment-card-empty";
import { useProjectData } from "../../data-provider";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { useAppEnvironment } from "../environment-context";
import { CreateDeploymentButton } from "../navigations/create-deployment-button";
import { ActiveBranches } from "./components/active-branches";
import { AppProductionCard } from "./components/app-production-card";
import { AppProductionCardSkeleton } from "./components/app-production-card-skeleton";
import { EnvironmentPendingCard } from "./components/environment-pending-card";
import { LatestDeployments } from "./components/latest-deployments";
import { OverviewHeader, ProductionActions } from "./components/overview-header";
import { ProductionCardProvider } from "./components/production-card-context";
import { useAppOverview } from "./components/use-app-overview";

const historySectionByKind: Record<EnvironmentKind, ComponentType> = {
  production: LatestDeployments,
  preview: ActiveBranches,
};

export default function Overview() {
  const { deployments, isDeploymentsLoading } = useProjectData();
  const { app } = useAppCurrentDeployment();
  const { environment } = useAppEnvironment();
  const overview = useAppOverview();
  const HistorySection =
    app?.sourceType === "oci" ? LatestDeployments : historySectionByKind[environment.kind];
  const hasNoDeployments = !isDeploymentsLoading && deployments.length === 0;

  const { header, card } = match(overview)
    .with({ kind: "loading" }, () => ({
      header: <OverviewHeader />,
      card: <AppProductionCardSkeleton />,
    }))
    .with({ kind: "pending" }, ({ newerDeployment }) => ({
      header: (
        <OverviewHeader
          actions={
            <CreateDeploymentButton
              renderTrigger={({ onClick }) => (
                <Button variant="outline" size="sm" onClick={onClick}>
                  <IconPlusOutline18 />
                  Create deployment
                </Button>
              )}
            />
          }
        />
      ),
      card: <EnvironmentPendingCard newerDeployment={newerDeployment} />,
    }))
    .with({ kind: "deployed" }, ({ card, domains, newerDeployment, dialogs }) => ({
      header: (
        <OverviewHeader
          deployment={card.deployment}
          primaryDomain={domains.at(0)}
          actions={<ProductionActions />}
        />
      ),
      card: (
        <>
          <AppProductionCard domains={domains} newerDeployment={newerDeployment} />
          {dialogs}
        </>
      ),
    }))
    .exhaustive();

  return (
    <ProductionCardProvider value={overview.kind === "deployed" ? overview.card : null}>
      <PageContainer>
        {hasNoDeployments ? <OverviewHeader /> : header}
        <PageBody>
          {hasNoDeployments ? (
            <CreateDeploymentButton
              renderTrigger={({ onClick }) => (
                <ActiveDeploymentCardEmpty onCreateDeployment={onClick} />
              )}
            />
          ) : (
            <div className="flex flex-col gap-5">
              {card}
              <HistorySection />
            </div>
          )}
        </PageBody>
      </PageContainer>
    </ProductionCardProvider>
  );
}
