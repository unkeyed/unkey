"use client";

import type { DeploymentStatus } from "@/lib/collections";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { useProjectData } from "../../../data-provider";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
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
      statusBadge={<DeploymentStatusBadge status={deploymentStatus} />}
    />
  );
}
