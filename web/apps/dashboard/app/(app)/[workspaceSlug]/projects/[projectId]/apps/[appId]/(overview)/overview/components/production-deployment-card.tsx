"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import dynamic from "next/dynamic";
import { useState } from "react";
import { ActiveDeploymentCardEmpty } from "../../../components/active-deployment-card/components/active-deployment-card-empty";
import { getDomainPriority } from "../../../components/domain-priority";
import { Card } from "../../components/card";
import { useAppId, useProjectData } from "../../data-provider";
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";
import { BuildInProgressChart, ProductionCardChart } from "./card-chart";
import { ProductionCardHeader } from "./card-header";
import { ProductionCardMetadata } from "./card-metadata";
import { ProductionCardRollbackBanner } from "./card-rollback-banner";
import { buildPulse } from "./g-pulse";
import { type ProductionCardContextValue, ProductionCardProvider } from "./production-card-context";
import { ProductionDeploymentCardSkeleton } from "./production-deployment-card-skeleton";
import { deriveProductionStatus } from "./status";

const RollbackDialog = dynamic(
  () =>
    import("../../deployments/components/table/components/actions/rollback-dialog").then(
      (m) => m.RollbackDialog,
    ),
  { ssr: false },
);

const UndoRollbackDialog = dynamic(
  () => import("./undo-rollback-dialog").then((m) => m.UndoRollbackDialog),
  { ssr: false },
);

export function ProductionDeploymentCard() {
  const {
    projectId,
    deployments,
    environments,
    customDomains,
    getDomainsForDeployment,
    isDeploymentsLoading,
  } = useProjectData();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [undoOpen, setUndoOpen] = useState(false);

  const appsQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appsQuery.data?.[0];
  const repoFullName = app?.repositoryFullName ?? null;

  const currentDeploymentId = app?.currentDeploymentId ?? null;
  const currentDeploymentQuery = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          and(
            eq(deployment.projectId, projectId),
            eq(deployment.appId, appId),
            eq(deployment.id, currentDeploymentId ?? ""),
          ),
        ),
    [projectId, appId, currentDeploymentId],
  );
  const currentDeployment = currentDeploymentId ? currentDeploymentQuery.data?.[0] : undefined;

  const productionEnvironmentId = environments.find((e) => e.slug === "production")?.id;
  const latestProductionDeployment = productionEnvironmentId
    ? deployments.find((d) => d.environmentId === productionEnvironmentId)
    : undefined;

  const deployment = currentDeployment ?? latestProductionDeployment;
  const isCurrent = Boolean(currentDeployment);

  const metrics = trpc.deploy.metrics.getAppRpsMetrics.useQuery(
    { appId },
    { refetchInterval: 30_000 },
  );

  const productionStatus = deployment ? deriveProductionStatus(deployment) : undefined;
  useCollectionPolling(() => collection.deployments.utils.refetch(), {
    intervalMs: 10_000,
    enabled: productionStatus === "live" || productionStatus === "crashing",
  });

  const isResolvingCurrentDeployment =
    currentDeploymentId != null && currentDeploymentQuery.isLoading;

  if (isDeploymentsLoading || appsQuery.isLoading || isResolvingCurrentDeployment) {
    return <ProductionDeploymentCardSkeleton />;
  }

  if (!deployment) {
    return (
      <CreateDeploymentButton
        renderTrigger={({ onClick }) => (
          <ActiveDeploymentCardEmpty
            onCreateDeployment={onClick}
            title="No production deployment yet"
            description="This app hasn't been deployed to production. Deploy to production to make it live."
          />
        )}
      />
    );
  }

  const status = productionStatus ?? deriveProductionStatus(deployment);
  const isRolledBack = isCurrent ? (app?.isRolledBack ?? false) : false;
  const sourceRepo = deployment.forkRepositoryFullName || repoFullName;

  const { primary, additional } = getDomainPriority({
    domains: getDomainsForDeployment(deployment.id),
    customDomains,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    currentDeploymentId,
  });

  const readySiblings = deployments.filter(
    (d) =>
      d.environmentId === deployment.environmentId &&
      d.status === "ready" &&
      d.id !== deployment.id,
  );
  const rollbackTarget = readySiblings
    .filter((d) => d.createdAt < deployment.createdAt)
    .sort((a, b) => b.createdAt - a.createdAt)[0];
  const undoCandidates = isRolledBack
    ? [...readySiblings, deployment].sort((a, b) => b.createdAt - a.createdAt)
    : [];
  const rolledBackFromDeployment = isRolledBack
    ? readySiblings
        .filter((d) => d.createdAt > deployment.createdAt)
        .sort((a, b) => b.createdAt - a.createdAt)[0]
    : undefined;

  const diagnostic =
    status === "crashing"
      ? {
          label: "View crash logs",
          href: routes.projects.logs({
            workspaceSlug: workspace.slug,
            projectId,
            appId,
            deploymentId: deployment.id,
          }),
        }
      : status === "failed"
        ? {
            label: "View build error",
            href: routes.projects.apps.deployment({
              workspaceSlug: workspace.slug,
              projectId,
              appId,
              deploymentId: deployment.id,
              build: true,
            }),
          }
        : null;

  const addCustomDomainHref =
    primary?.source === "platform"
      ? {
          pathname: routes.projects.apps.settings({
            workspaceSlug: workspace.slug,
            projectId,
            appId,
          }),
          hash: "custom-domains",
        }
      : null;

  const ctx: ProductionCardContextValue = {
    deployment,
    status,
    isCurrent,
    isRolledBack,
    rolledBackFrom: rolledBackFromDeployment
      ? {
          commitSha: rolledBackFromDeployment.gitCommitSha,
          commitMessage: rolledBackFromDeployment.gitCommitMessage,
        }
      : null,
    sourceRepo,
    primaryDomain: primary ? { hostname: primary.hostname, url: primary.url } : null,
    additionalDomains: additional.map((d) => ({ hostname: d.hostname, url: d.url })),
    addCustomDomainHref,
    diagnostic,
    logsHref: routes.projects.logs({ workspaceSlug: workspace.slug, projectId, appId }),
    requestsHref: routes.projects.requests({
      workspaceSlug: workspace.slug,
      projectId,
      appId,
      since: "6h",
    }),
    rollbackTarget,
    undoCandidates,
    pulse: buildPulse(metrics.data),
    isChartLoading: metrics.isLoading,
    isChartError: metrics.isError,
    openRollback: () => setRollbackOpen(true),
    openUndo: () => setUndoOpen(true),
  };

  return (
    <ProductionCardProvider value={ctx}>
      <div className="relative">
        {isRolledBack && <ProductionCardRollbackBanner />}
        <Card className="relative z-10 flex flex-col">
          <ProductionCardHeader />
          <div className="grid grid-cols-1 md:grid-cols-2">
            {!isCurrent && status === "deploying" ? (
              <BuildInProgressChart />
            ) : (
              <ProductionCardChart />
            )}
            <div className="p-4">
              <ProductionCardMetadata />
            </div>
          </div>
        </Card>
      </div>

      {rollbackTarget && (
        <RollbackDialog
          isOpen={rollbackOpen}
          onClose={() => setRollbackOpen(false)}
          targetDeployment={rollbackTarget}
          currentDeployment={deployment}
        />
      )}
      {isRolledBack && undoCandidates.length > 0 && (
        <UndoRollbackDialog
          isOpen={undoOpen}
          onClose={() => setUndoOpen(false)}
          deployments={undoCandidates}
          currentDeploymentId={deployment.id}
        />
      )}
    </ProductionCardProvider>
  );
}
