"use client";

import type { DeploymentStatus } from "@/lib/collections";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusLabel } from "../../../../components/deployment-status-dot";
import { useProjectData } from "../../../data-provider";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { InstantRollbackRow } from "../instant-rollback-row";
import { useDeployment } from "../layout-provider";

type DeploymentInfoProps = {
  statusOverride?: DeploymentStatus;
};

export function DeploymentInfo({ statusOverride }: DeploymentInfoProps) {
  const { deployment } = useDeployment();
  const { environments } = useProjectData();
  const { app, isRolledBack: appIsRolledBack } = useAppCurrentDeployment();
  const deploymentStatus = statusOverride ?? deployment.status;

  const isCurrent = app?.currentDeploymentId === deployment.id;
  const isRolledBack = isCurrent && appIsRolledBack;
  const environment = environments.find((e) => e.id === deployment.environmentId);

  return (
    <ActiveDeploymentCard
      deploymentId={deployment.id}
      deployment={deployment}
      isCurrent={isCurrent}
      isRolledBack={isRolledBack}
      environmentSlug={environment?.slug}
      statusBadge={<DeploymentStatusLabel status={deploymentStatus} className="shrink-0 text-xs" />}
      expandableContent={<InstantRollbackRow key={deployment.id} />}
    />
  );
}
