"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { type Deployment, collection } from "@/lib/collections";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import {
  findRolledBackFrom,
  previousRollbackTarget,
  rollbackCandidates,
} from "@/lib/collections/deploy/rollback";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import dynamic from "next/dynamic";
import { type ReactNode, useState } from "react";
import { getDomainPriority } from "../../../components/domain-priority";
import { useProjectData } from "../../../data-provider";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppEnvironment, useAppScope } from "../../environment-context";
import { hasVisibleBuildState } from "./card-newer-deployment";
import { buildPulse } from "./g-pulse";
import type { CardDomain, ProductionCardContextValue } from "./production-card-context";
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

export type AppOverview =
  | { kind: "loading" }
  | { kind: "pending"; newerDeployment: Deployment | undefined }
  | {
      kind: "deployed";
      card: ProductionCardContextValue;
      domains: CardDomain[];
      newerDeployment: Deployment | undefined;
      dialogs: ReactNode;
    };

export function useAppOverview(): AppOverview {
  const {
    deployments,
    domains,
    customDomains,
    isDeploymentsLoading,
    isDomainsLoading,
    getLiveDomains,
  } = useProjectData();
  const scope = useAppScope();
  const { projectId, appId } = scope;
  const { environment } = useAppEnvironment();
  const { gated, openPaywall, planGate } = useDeployActionGate();
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

  // Production shows what serves traffic; any other environment shows its
  // latest ready deployment, since nothing there is promoted.
  const isProduction = environment.kind === ENVIRONMENT_KIND.production;
  const environmentDeployments = deployments.filter((d) => d.environmentId === environment.id);
  const latest = environmentDeployments.at(0);
  const deployment = isProduction
    ? (currentDeployment ?? latest)
    : environmentDeployments.find((d) => d.status === "ready");
  const [rollbackTargetSnapshot, setRollbackTargetSnapshot] = useState<Deployment>();
  const isCurrent = isProduction && Boolean(currentDeployment);
  const newerDeployment =
    latest && latest.id !== deployment?.id && hasVisibleBuildState(latest) ? latest : undefined;

  const metrics = trpc.deploy.metrics.getAppRpsMetrics.useQuery(
    { appId },
    { refetchInterval: 30_000 },
  );

  const productionStatus = deployment ? deriveProductionStatus(deployment) : undefined;
  useCollectionPolling(() => collection.deployments.utils.refetch(), {
    intervalMs: 10_000,
    enabled:
      productionStatus === "ready" ||
      productionStatus === "crashing" ||
      productionStatus === "deploying" ||
      (newerDeployment ? isDeploymentInFlight(newerDeployment.status) : false),
  });

  if (isDeploymentsLoading || isCurrentDeploymentLoading || isDomainsLoading) {
    return { kind: "loading" };
  }

  if (!deployment) {
    return { kind: "pending", newerDeployment };
  }

  const status = productionStatus ?? deriveProductionStatus(deployment);
  const isRolledBack = isCurrent && appIsRolledBack;
  const sourceRepo = deployment.forkRepositoryFullName || repoFullName;

  const { primary, additional } = getDomainPriority({
    domains: isProduction
      ? getLiveDomains()
      : domains.filter((d) => d.deploymentId === deployment.id),
    customDomains,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    currentDeploymentId,
  });

  const rollbackTarget = isCurrent ? previousRollbackTarget(deployments, deployment) : undefined;
  const undoCandidates = isRolledBack
    ? [...rollbackCandidates(deployments, deployment), deployment].sort(
        (a, b) => b.createdAt - a.createdAt,
      )
    : [];
  const rolledBackFromDeployment = isRolledBack
    ? findRolledBackFrom(deployments, deployment)
    : undefined;

  const diagnostic =
    status === "crashing"
      ? {
          label: "View crash logs",
          href: routes.projects.logs({
            workspaceSlug: scope.workspaceSlug,
            projectId,
            appId,
            deploymentId: deployment.id,
          }),
        }
      : status === "failed"
        ? {
            label: "View build error",
            href: routes.projects.apps.deployment({
              ...scope,
              deploymentId: deployment.id,
              build: true,
            }),
          }
        : null;

  const card: ProductionCardContextValue = {
    deployment,
    status,
    isCurrent,
    isRolledBack,
    rolledBackFrom: rolledBackFromDeployment
      ? {
          commitSha:
            rolledBackFromDeployment.source === "git"
              ? rolledBackFromDeployment.gitCommitSha
              : null,
          commitMessage:
            rolledBackFromDeployment.source === "git"
              ? rolledBackFromDeployment.gitCommitMessage
              : null,
          image:
            rolledBackFromDeployment.source === "oci"
              ? (rolledBackFromDeployment.requestedImage ?? rolledBackFromDeployment.resolvedImage)
              : null,
        }
      : null,
    sourceRepo,
    diagnostic,
    deploymentHref: routes.projects.apps.deployment({ ...scope, deploymentId: deployment.id }),
    logsHref: routes.projects.logs({
      workspaceSlug: scope.workspaceSlug,
      projectId,
      appId,
      environmentId: environment.id,
    }),
    requestsHref: routes.projects.requests({
      workspaceSlug: scope.workspaceSlug,
      projectId,
      appId,
      environmentId: environment.id,
      since: "6h",
    }),
    rollbackTarget,
    undoCandidates,
    pulse: buildPulse(metrics.data),
    isChartLoading: metrics.isLoading,
    isChartError: metrics.isError,
    // Without a Compute plan these open the paywall instead of switching traffic.
    openRollback: () => {
      if (gated) {
        openPaywall();
        return;
      }
      setRollbackTargetSnapshot(rollbackTarget);
      setRollbackOpen(true);
    },
    openUndo: () => (gated ? openPaywall() : setUndoOpen(true)),
  };

  return {
    kind: "deployed",
    card,
    domains: primary ? [primary, ...additional] : [],
    newerDeployment,
    dialogs: (
      <>
        {rollbackOpen && rollbackTargetSnapshot && (
          <RollbackDialog
            isOpen={rollbackOpen}
            onClose={() => setRollbackOpen(false)}
            targetDeployment={rollbackTargetSnapshot}
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
        {planGate}
      </>
    ),
  };
}
