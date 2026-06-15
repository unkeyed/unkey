"use client";

import type { DeploymentStatus } from "@/lib/collections";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";

type DeploymentInfoProps = {
  statusOverride?: DeploymentStatus;
};

export function DeploymentInfo({ statusOverride }: DeploymentInfoProps) {
  const { deployment } = useDeployment();
  const { project, environments } = useProjectData();
  const deploymentStatus = statusOverride ?? deployment.status;

  const isCurrent = project?.currentDeploymentId === deployment.id;
  const isRolledBack = isCurrent && (project?.isRolledBack ?? false);
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
