"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { appSource } from "@/components/deploy/app-source";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type Deployment, type DeploymentSummary, collection } from "@/lib/collections";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { previousRollbackTarget, rollbackCandidates } from "@/lib/collections/deploy/rollback";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { match } from "@unkey/match";
import { Card, PageBody } from "@unkey/ui";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import type { Route } from "next";
import dynamic from "next/dynamic";
import { type ReactNode, useState } from "react";
import { getDomainPriority } from "../../../components/domain-priority";
import { useAppId, useProjectData } from "../../data-provider";
import { appDeploymentQueryFor } from "../../data-provider-queries";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { useRecentDeployments } from "../../hooks/use-recent-deployments";
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";
import { ActiveDeploymentCardEmpty } from "./active-deployment-card-empty";
import { AppCanvas } from "./app-canvas";
import { AppCanvasSkeleton } from "./app-canvas-skeleton";
import { NewerDeploymentRow, hasVisibleBuildState } from "./card-newer-deployment";
import { ProductionCardRollbackBanner } from "./card-rollback-banner";
import { OverviewHeader, ProductionActions, ProductionActionsSkeleton } from "./overview-header";
import { type ProductionCardContextValue, ProductionCardProvider } from "./production-card-context";
import { STATUS_META, deriveProductionStatus } from "./status";

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

type ProductionOverviewModel =
  | { kind: "loading" }
  | { kind: "no-deployments" }
  | { kind: "no-production" }
  | {
      kind: "deployed";
      card: ProductionCardContextValue;
      newer: {
        deployment: DeploymentSummary & Pick<Deployment, "buildEndedAt">;
        href: Route;
        actionLabel?: string;
      } | null;
      rollbackDialog: { target: DeploymentSummary; close: () => void } | null;
      undoDialog: { open: boolean; close: () => void } | null;
    };

export function ProductionOverview({ children }: { children: ReactNode }) {
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const model = useProductionOverview({ gated, openPaywall });
  return (
    <>
      <ProductionOverviewView model={model}>{children}</ProductionOverviewView>
      {planGate}
    </>
  );
}

function useProductionOverview({
  gated,
  openPaywall,
}: {
  gated: boolean;
  openPaywall: () => void;
}): ProductionOverviewModel {
  const { projectId, environments, customDomains } = useProjectData();
  const { deployments, isLoading: isDeploymentsLoading } = useRecentDeployments();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [undoOpen, setUndoOpen] = useState(false);

  const {
    app,
    currentDeployment,
    isRolledBack: appIsRolledBack,
    isLoading: isCurrentDeploymentLoading,
  } = useAppCurrentDeployment();
  const repoFullName = app?.repositoryFullName ?? null;
  const currentDeploymentId = app?.currentDeploymentId ?? null;

  const productionEnvironmentId = environments.find(
    (e) => e.kind === ENVIRONMENT_KIND.production,
  )?.id;
  const latestProductionDeployment = productionEnvironmentId
    ? deployments.find((d) => d.environmentId === productionEnvironmentId)
    : undefined;
  // Without a live deployment the card shows the latest production one, which
  // needs its instances, so it loads by id like the current deployment
  const fallbackDeploymentId =
    !currentDeploymentId && latestProductionDeployment ? latestProductionDeployment.id : null;
  const fallbackDeploymentQuery = useLiveQuery(
    (q) =>
      fallbackDeploymentId
        ? appDeploymentQueryFor(projectId, appId, fallbackDeploymentId)(q)
        : null,
    [projectId, appId, fallbackDeploymentId],
  );

  const deployment = currentDeployment ?? fallbackDeploymentQuery.data?.[0];
  const [rollbackTargetSnapshot, setRollbackTargetSnapshot] = useState<DeploymentSummary>();
  const isCurrent = Boolean(currentDeployment);
  const newerDeployment =
    deployment &&
    latestProductionDeployment &&
    latestProductionDeployment.id !== deployment.id &&
    hasVisibleBuildState(latestProductionDeployment)
      ? latestProductionDeployment
      : undefined;
  const liveDomainsQuery = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) =>
          and(eq(domain.projectId, projectId), eq(domain.appId, appId), eq(domain.sticky, "live")),
        ),
    [projectId, appId],
  );

  const productionStatus = deployment ? deriveProductionStatus(deployment) : undefined;
  useCollectionPolling(() => collection.deployments.utils.refetch(), {
    intervalMs: 10_000,
    enabled:
      (productionStatus ? STATUS_META[productionStatus].polls : false) ||
      (newerDeployment ? isDeploymentInFlight(newerDeployment.status) : false),
  });

  if (!isDeploymentsLoading && deployments.length === 0) {
    return { kind: "no-deployments" };
  }

  if (
    isDeploymentsLoading ||
    isCurrentDeploymentLoading ||
    fallbackDeploymentQuery.isLoading ||
    liveDomainsQuery.isLoading
  ) {
    return { kind: "loading" };
  }

  if (!deployment) {
    return { kind: "no-production" };
  }

  const status = productionStatus ?? deriveProductionStatus(deployment);
  const isRolledBack = isCurrent && appIsRolledBack;
  const sourceRepo = deployment.forkRepositoryFullName || repoFullName;

  const { all } = getDomainPriority({
    domains: liveDomainsQuery.data ?? [],
    customDomains,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    currentDeploymentId,
  });

  const rollbackTarget = previousRollbackTarget(deployments, deployment);
  const undoCandidates: DeploymentSummary[] = isRolledBack
    ? [...rollbackCandidates(deployments, deployment), deployment].sort(
        (a, b) => b.createdAt - a.createdAt,
      )
    : [];

  const deploymentHref = routes.projects.apps.deployment({
    workspaceSlug: workspace.slug,
    projectId,
    appId,
    deploymentId: deployment.id,
  });
  const diagnostic = match(status)
    .returnType<ProductionCardContextValue["diagnostic"]>()
    .with("crashing", () => ({
      label: "View crash logs",
      href: routes.projects.logs({
        workspaceSlug: workspace.slug,
        projectId,
        appId,
        deploymentId: deployment.id,
      }),
    }))
    .with("failed", () => ({ label: "View error", href: deploymentHref }))
    .otherwise(() => null);

  const appScope = { workspaceSlug: workspace.slug, projectId, appId };
  const newerFailed = newerDeployment?.status === "failed";

  const ctx: ProductionCardContextValue = {
    deployment,
    status,
    isCurrent,
    isRolledBack,
    canvas: {
      appName: app?.name ?? "App",
      appSource: app ? appSource(app) : "legacy",
      defaultBranch: app?.defaultBranch ?? null,
      domains: all,
      domainsHref: routes.projects.apps.domains(appScope),
      addDomainHref:
        isCurrent && !all.some((d) => d.source === "custom")
          ? routes.projects.apps.domains({ ...appScope, add: true })
          : null,
      policiesHref: routes.projects.apps.policies(appScope),
      addPolicyHref: routes.projects.apps.policies({ ...appScope, add: true }),
      connectionsHref: routes.projects.apps.connections(appScope),
      addConnectionHref: routes.projects.apps.connections({ ...appScope, add: true }),
    },
    sourceRepo,
    diagnostic,
    deploymentHref,
    logsHref: routes.projects.logs({ workspaceSlug: workspace.slug, projectId, appId }),
    requestsHref: routes.projects.requests({
      workspaceSlug: workspace.slug,
      projectId,
      appId,
      since: "6h",
    }),
    rollbackTarget,
    undoCandidates,
    // Without a Compute plan these open the paywall instead of switching traffic.
    openRollback: () => {
      if (gated) {
        openPaywall();
        return;
      }
      setRollbackTargetSnapshot(rollbackTarget);
      setRollbackOpen(true);
    },
    openUndo: () => {
      if (gated) {
        openPaywall();
        return;
      }
      setUndoOpen(true);
    },
  };

  return {
    kind: "deployed",
    card: ctx,
    newer: newerDeployment
      ? {
          deployment: newerDeployment,
          href: routes.projects.apps.deployment({
            ...appScope,
            deploymentId: newerDeployment.id,
            build: !newerFailed,
          }),
          actionLabel: newerFailed ? "View error" : undefined,
        }
      : null,
    rollbackDialog:
      rollbackOpen && rollbackTargetSnapshot
        ? { target: rollbackTargetSnapshot, close: () => setRollbackOpen(false) }
        : null,
    undoDialog:
      isRolledBack && undoCandidates.length > 0
        ? { open: undoOpen, close: () => setUndoOpen(false) }
        : null,
  };
}

function ProductionOverviewView({
  model,
  children,
}: {
  model: ProductionOverviewModel;
  children: ReactNode;
}) {
  const deployed = model.kind === "deployed" ? model : null;
  return (
    <ProductionCardProvider value={deployed?.card ?? null}>
      <OverviewHeader
        actions={match(model.kind)
          .with("deployed", () => <ProductionActions />)
          .with("loading", () => <ProductionActionsSkeleton />)
          .with("no-deployments", "no-production", () => undefined)
          .exhaustive()}
      />
      <PageBody>
        <div className="flex flex-col gap-5">
          <ProductionCardSlot model={model} />
          {model.kind !== "no-deployments" && children}
        </div>
      </PageBody>
      {deployed?.rollbackDialog && (
        <RollbackDialog
          isOpen
          onClose={deployed.rollbackDialog.close}
          targetDeployment={deployed.rollbackDialog.target}
          currentDeployment={deployed.card.deployment}
        />
      )}
      {deployed?.undoDialog && (
        <UndoRollbackDialog
          isOpen={deployed.undoDialog.open}
          onClose={deployed.undoDialog.close}
          deployments={deployed.card.undoCandidates}
          currentDeploymentId={deployed.card.deployment.id}
        />
      )}
    </ProductionCardProvider>
  );
}

function ProductionCardSlot({ model }: { model: ProductionOverviewModel }) {
  const reduceMotion = useReducedMotion();
  return match(model)
    .with({ kind: "loading" }, () => <AppCanvasSkeleton />)
    .with({ kind: "no-deployments" }, () => (
      <CreateDeploymentButton
        renderTrigger={({ onClick }) => <ActiveDeploymentCardEmpty onCreateDeployment={onClick} />}
      />
    ))
    .with({ kind: "no-production" }, () => (
      <CreateDeploymentButton
        renderTrigger={({ onClick }) => (
          <ActiveDeploymentCardEmpty
            onCreateDeployment={onClick}
            title="No production deployment yet"
            description="This app hasn't been deployed to production. Deploy to production to make it live."
          />
        )}
      />
    ))
    .with({ kind: "deployed" }, ({ card, newer }) => (
      <div className="relative">
        {card.isRolledBack && <ProductionCardRollbackBanner />}
        <Card className="relative z-10 flex flex-col">
          <AppCanvas />
          <AnimatePresence initial={false} mode="wait">
            {newer && (
              <motion.div
                key={newer.deployment.id}
                className="overflow-hidden"
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: "auto", opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={
                  reduceMotion ? { duration: 0 } : { duration: 0.2, ease: [0.215, 0.61, 0.355, 1] }
                }
              >
                <NewerDeploymentRow {...newer} />
              </motion.div>
            )}
          </AnimatePresence>
        </Card>
      </div>
    ))
    .exhaustive();
}
