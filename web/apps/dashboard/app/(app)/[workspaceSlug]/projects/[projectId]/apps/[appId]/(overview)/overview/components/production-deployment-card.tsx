"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import type { LastExit } from "@/lib/types/deploy";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Button, DialogContainer } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useEffect, useState } from "react";
import { ActiveDeploymentCardEmpty } from "../../../components/active-deployment-card/components/active-deployment-card-empty";
import { getDomainPriority } from "../../../components/domain-priority";
import { useAppId, useProjectData } from "../../data-provider";
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";
import { adaptiveWindowForAge, buildPulse } from "./g-pulse";
import { ProductionDeploymentCardSkeleton } from "./production-deployment-card-skeleton";
import {
  type DeploymentDisplayStatus,
  ProductionDeploymentCardView,
  type ProductionDeploymentViewModel,
} from "./production-deployment-card-view";
import { useOverviewDebug } from "./use-overview-debug";

const RollbackDialog = dynamic(
  () =>
    import("../../deployments/components/table/components/actions/rollback-dialog").then(
      (m) => m.RollbackDialog,
    ),
  { ssr: false },
);

const SYNTHETIC_LAST_EXIT: LastExit = {
  restartCount: 5,
  exitCode: 1,
  signal: null,
  reason: "CrashLoopBackOff",
  finishedAt: null,
  statusReason: "Back-off restarting failed container",
};

export function ProductionDeploymentCard() {
  const {
    project,
    projectId,
    deployments,
    customDomains,
    getDeploymentById,
    getDomainsForDeployment,
    isDeploymentsLoading,
  } = useProjectData();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [undoOpen, setUndoOpen] = useState(false);
  const debug = useOverviewDebug();

  // Adaptive window keys off the deployment's age. Computed after mount so the
  // server and first client render agree (both fall back to "week").
  const [now, setNow] = useState<number | null>(null);
  useEffect(() => setNow(Date.now()), []);

  const currentDeploymentId = project?.currentDeploymentId ?? null;

  const appsQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  // Synthetic fallback so the Source links render as GitHub links on repo-less
  // local apps. Drop the fallback when wiring live (same pass as the debug nav).
  const repoFullName = appsQuery.data?.[0]?.repositoryFullName ?? "unkey/unkey";

  const deployment = currentDeploymentId ? getDeploymentById(currentDeploymentId) : undefined;

  if (isDeploymentsLoading || debug.view === "loading") {
    return <ProductionDeploymentCardSkeleton />;
  }

  if (!deployment || debug.view === "empty") {
    return (
      <CreateDeploymentButton
        renderTrigger={({ onClick }) => <ActiveDeploymentCardEmpty onCreateDeployment={onClick} />}
      />
    );
  }

  const { primary, additional } = getDomainPriority({
    domains: getDomainsForDeployment(deployment.id),
    customDomains,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    currentDeploymentId,
  });

  const rollbackTarget = deployments.find(
    (d) =>
      d.id !== deployment.id &&
      d.environmentId === deployment.environmentId &&
      d.status === "ready" &&
      d.createdAt < deployment.createdAt,
  );

  const instances = deployment.instances ?? [];
  const regionSource =
    instances.length > 0
      ? [...new Map(instances.map((i) => [i.region.id, i])).values()]
      : deployment.desiredRegions;

  // --- debug overrides (everything below is synthetic this pass) ---
  const status: DeploymentDisplayStatus = debug.status;

  const realPrimary = primary ? { hostname: primary.hostname, url: primary.url } : null;
  // Live: show the nudge when the primary domain is platform-generated and no
  // custom domain exists (primary.source === "platform"). Synthetic: keyed off
  // the debug "generated" state.
  const addCustomDomainHref = `${routes.projects.apps.settings({
    workspaceSlug: workspace.slug,
    projectId,
    appId,
  })}#custom-domains`;
  const domainView =
    debug.domain === "none"
      ? { primaryDomain: null, additionalDomainCount: 0, addCustomDomainHref: undefined }
      : debug.domain === "generated"
        ? {
            primaryDomain: {
              hostname: "app-rhtktlljk.unkey.app",
              url: "https://app-rhtktlljk.unkey.app",
            },
            additionalDomainCount: 0,
            addCustomDomainHref,
          }
        : {
            primaryDomain: realPrimary ?? {
              hostname: "api.unkey.com",
              url: "https://api.unkey.com",
            },
            additionalDomainCount: realPrimary ? additional.length : 2,
            addCustomDomainHref: undefined,
          };

  // Crashing → runtime logs for this deployment; failed (build) → build logs.
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
        : undefined;

  const windowKey =
    debug.win === "auto"
      ? now
        ? adaptiveWindowForAge(now - deployment.createdAt)
        : "week"
      : debug.win;
  const pulse = buildPulse(windowKey, debug.traffic);

  const vm: ProductionDeploymentViewModel = {
    status,
    rolledBack: debug.rolledBack,
    // Synthetic superseded deployment. Live: query the most recent
    // status="superseded" deployment in this environment.
    rolledBackFrom: debug.rolledBack
      ? {
          commitSha: "833a8ff",
          commitMessage: "chore(nuke): stop and remove containers before pruning",
        }
      : null,
    branch: deployment.gitBranch,
    commitSha: deployment.gitCommitSha,
    commitMessage: deployment.gitCommitMessage,
    image: deployment.image,
    repoFullName,
    forkRepositoryFullName: deployment.forkRepositoryFullName,
    authorHandle: deployment.gitCommitAuthorHandle,
    authorAvatarUrl: deployment.gitCommitAuthorAvatarUrl,
    createdAt: deployment.createdAt,
    primaryDomain: domainView.primaryDomain,
    additionalDomainCount: domainView.additionalDomainCount,
    addCustomDomainHref: domainView.addCustomDomainHref,
    canRollback: Boolean(rollbackTarget),
    regions: regionSource.map((r) => ({
      id: r.region.id,
      name: r.region.name,
      flagCode: r.flagCode,
    })),
    runningCount: status === "stopped" ? 0 : instances.filter((i) => i.status === "running").length,
    targetCount: deployment.desiredInstanceCount,
    cpuMillicores: deployment.cpuMillicores,
    memoryMib: deployment.memoryMib,
    storageMib: deployment.storageMib,
    // Runtime crash detail only applies to a crash-loop; a build "failed" never
    // ran a container, so no lastExit badge there.
    lastExit: status === "crashing" ? (deployment.lastExit ?? SYNTHETIC_LAST_EXIT) : null,
    logsHref: routes.projects.logs({ workspaceSlug: workspace.slug, projectId, appId }),
    requestsHref: routes.projects.requests({
      workspaceSlug: workspace.slug,
      projectId,
      appId,
      since: "6h",
    }),
    diagnostic,
  };

  return (
    <>
      <ProductionDeploymentCardView
        vm={vm}
        pulse={pulse}
        onRollback={rollbackTarget ? () => setRollbackOpen(true) : undefined}
        onUndoRollback={debug.rolledBack ? () => setUndoOpen(true) : undefined}
      />
      {rollbackTarget && (
        <RollbackDialog
          isOpen={rollbackOpen}
          onClose={() => setRollbackOpen(false)}
          targetDeployment={rollbackTarget}
          currentDeployment={deployment}
        />
      )}
      <DialogContainer
        isOpen={undoOpen}
        onOpenChange={() => setUndoOpen(false)}
        title="Undo rollback?"
        subTitle="Resume automatic production deploys"
        footer={
          <Button
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
            onClick={() => {
              // Synthetic undo. Live: call the undo/promote mutation to restore
              // the superseded deployment and unset isRolledBack.
              debug.setRolledBack(false);
              setUndoOpen(false);
            }}
          >
            Undo rollback
          </Button>
        }
      >
        <p className="text-[13px] text-gray-11">
          This re-enables automatic production deploys. The currently live deployment stays live
          until your next deploy or rollback.
        </p>
      </DialogContainer>
    </>
  );
}
