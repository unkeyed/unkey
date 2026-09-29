"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { collection } from "@/lib/collections";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { findRolledBackFrom } from "@/lib/collections/deploy/rollback";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Card } from "@unkey/ui";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import dynamic from "next/dynamic";
import { useState } from "react";
import { getDomainPriority } from "../../../components/domain-priority";
import { useProjectData } from "../../../data-provider";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppEnvironment, useAppScope } from "../../environment-context";
import { AddDomainGhost, AppCanvas, AppNode } from "./app-canvas";
import { AppProductionCardSkeleton } from "./app-production-card-skeleton";
import { ProductionCardHeader } from "./card-header";
import { NewerDeploymentRow, hasVisibleBuildState } from "./card-newer-deployment";
import { ProductionCardRollbackBanner } from "./card-rollback-banner";
import { EnvironmentPendingCard } from "./environment-pending-card";
import { buildPulse } from "./g-pulse";
import { type ProductionCardContextValue, ProductionCardProvider } from "./production-card-context";
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

export function AppProductionCard() {
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
  const reduceMotion = useReducedMotion();
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
      productionStatus === "live" ||
      productionStatus === "crashing" ||
      productionStatus === "deploying" ||
      (newerDeployment ? isDeploymentInFlight(newerDeployment.status) : false),
  });

  if (isDeploymentsLoading || isCurrentDeploymentLoading || isDomainsLoading) {
    return <AppProductionCardSkeleton />;
  }

  if (!deployment) {
    return <EnvironmentPendingCard newerDeployment={newerDeployment} />;
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

  const addCustomDomainHref =
    primary?.source === "platform"
      ? {
          pathname: routes.projects.apps.settings(scope),
          hash: "custom-domains",
        }
      : null;

  const ctx: ProductionCardContextValue = {
    eyebrow: isProduction ? null : `Latest ${environment.slug}`,
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
    primaryDomain: primary
      ? { hostname: primary.hostname, url: primary.url, source: primary.source }
      : null,
    additionalDomains: additional.map((d) => ({
      hostname: d.hostname,
      url: d.url,
      source: d.source,
    })),
    addCustomDomainHref,
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
    openRollback: () => (gated ? openPaywall() : setRollbackOpen(true)),
    openUndo: () => (gated ? openPaywall() : setUndoOpen(true)),
  };

  return (
    <ProductionCardProvider value={ctx}>
      <div className="relative">
        {isRolledBack && <ProductionCardRollbackBanner />}
        <Card className="relative z-10 flex flex-col">
          <ProductionCardHeader />
          <AppCanvas
            domains={primary ? [primary, ...additional] : []}
            emptyDomain={<AddDomainGhost />}
            app={<AppNode />}
          />
          <AnimatePresence initial={false} mode="wait">
            {newerDeployment && (
              <motion.div
                key={newerDeployment.id}
                className="overflow-hidden"
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: "auto", opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={
                  reduceMotion ? { duration: 0 } : { duration: 0.2, ease: [0.215, 0.61, 0.355, 1] }
                }
              >
                <NewerDeploymentRow
                  deployment={newerDeployment}
                  href={routes.projects.apps.deployment({
                    ...scope,
                    deploymentId: newerDeployment.id,
                    build: true,
                  })}
                />
              </motion.div>
            )}
          </AnimatePresence>
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
      {planGate}
    </ProductionCardProvider>
  );
}
