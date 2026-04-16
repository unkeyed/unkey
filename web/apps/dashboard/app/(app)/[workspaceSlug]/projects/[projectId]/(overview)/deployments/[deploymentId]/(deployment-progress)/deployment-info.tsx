"use client";

import type { DeploymentStatus } from "@/lib/collections";
import { Cloud } from "@unkey/icons";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { Section, SectionHeader } from "../../../../components/section";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";

type DeploymentInfoProps = {
  title?: string;
  statusOverride?: DeploymentStatus;
};

export function DeploymentInfo({ title = "Deployment", statusOverride }: DeploymentInfoProps) {
  const { deployment } = useDeployment();
  const { project, environments } = useProjectData();
  const deploymentStatus = statusOverride ?? deployment.status;

  const isCurrent = project?.currentDeploymentId === deployment.id;
  const isRolledBack = isCurrent && (project?.isRolledBack ?? false);
  const environment = environments.find((e) => e.id === deployment.environmentId);

  return (
    <Section>
      <SectionHeader icon={<Cloud iconSize="md-regular" className="text-gray-9" />} title={title} />
      <ActiveDeploymentCard
        deploymentId={deployment.id}
        isCurrent={isCurrent}
        isRolledBack={isRolledBack}
        environmentSlug={environment?.slug}
        statusBadge={<DeploymentStatusBadge status={deploymentStatus} />}
      />
    </Section>
  );
}
