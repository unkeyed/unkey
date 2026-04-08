"use client";

import type { DeploymentStatus } from "@/lib/collections";
import {
  formatCpuParts,
  formatMemoryParts,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import { Bolt, Cloud, Grid, Harddrive, LayoutRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { InfoChip } from "../../../../components/info-chip";
import { RegionFlags } from "../../../../components/region-flags";
import { Section, SectionHeader } from "../../../../components/section";
import { useOptionalProjectLayout } from "../../../layout-provider";
import { useDeployment } from "../layout-provider";

type DeploymentInfoProps = {
  title?: string;
  statusOverride?: DeploymentStatus;
};

export function DeploymentInfo({ title = "Deployment", statusOverride }: DeploymentInfoProps) {
  const { deployment } = useDeployment();
  const projectLayout = useOptionalProjectLayout();
  const deploymentStatus = statusOverride ?? deployment.status;

  return (
    <Section>
      <SectionHeader icon={<Cloud iconSize="md-regular" className="text-gray-9" />} title={title} />
      <ActiveDeploymentCard
        deploymentId={deployment.id}
        trailingContent={
          <div className="flex gap-1.5 items-center">
            <div className="2xl:flex gap-1.5 items-center hidden">
              <InfoChip icon={Bolt}>
                <div className="text-xs flex gap-0.5">
                  <span className="font-medium text-gray-12">
                    {formatCpuParts(deployment.cpuMillicores).value}
                  </span>
                  <span className="text-gray-11">
                    {formatCpuParts(deployment.cpuMillicores).unit}
                  </span>
                </div>
              </InfoChip>
              <InfoChip icon={Grid}>
                <div className="text-xs flex gap-0.5">
                  <span className="font-medium text-gray-12">
                    {formatMemoryParts(deployment.memoryMib).value}
                  </span>
                  <span className="text-gray-11">
                    {formatMemoryParts(deployment.memoryMib).unit}
                  </span>
                </div>
              </InfoChip>
              {deployment.storageMib > 0 && (
                <InfoChip icon={Harddrive}>
                  <div className="text-xs flex gap-0.5">
                    <span className="font-medium text-gray-12">
                      {formatStorageParts(deployment.storageMib).value}
                    </span>
                    <span className="text-gray-11">
                      {formatStorageParts(deployment.storageMib).unit}
                    </span>
                  </div>
                </InfoChip>
              )}
            </div>
            <RegionFlags instances={deployment.instances} />
            {projectLayout && (
              <InfoTooltip asChild content="Show deployment details">
                <Button
                  variant="ghost"
                  className="[&_svg]:size-3 size-3 rounded-sm"
                  size="icon"
                  onClick={() => projectLayout.setIsDetailsOpen(!projectLayout.isDetailsOpen)}
                >
                  <LayoutRight iconSize="sm-regular" className="text-gray-10" />
                </Button>
              </InfoTooltip>
            )}
          </div>
        }
        statusBadge={<DeploymentStatusBadge status={deploymentStatus} />}
      />
    </Section>
  );
}
