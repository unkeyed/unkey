"use client";

import { formatCpu, formatMemory } from "@/lib/utils/deployment-formatters";
import { Bolt, Cloud, Grid, Harddrive, LayoutRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { ActiveDeploymentCard } from "../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { DisabledWrapper } from "../../../../components/disabled-wrapper";
import { InfoChip } from "../../../../components/info-chip";
import { RegionFlags } from "../../../../components/region-flags";
import { Section, SectionHeader } from "../../../../components/section";
import { useOptionalProjectLayout } from "../../../layout-provider";
import { useDeployment } from "../layout-provider";

export function DeploymentInfo({ title = "Deployment" }: { title?: string }) {
  const { deployment } = useDeployment();
  const projectLayout = useOptionalProjectLayout();
  const deploymentStatus = deployment.status;

  return (
    <Section>
      <SectionHeader icon={<Cloud iconSize="md-regular" className="text-gray-9" />} title={title} />
      <ActiveDeploymentCard
        deploymentId={deployment.id}
        trailingContent={
          <div className="flex gap-1.5 items-center">
            <div className="2xl:flex gap-1.5 items-center hidden">
              <InfoChip icon={Bolt}>
                <div className="text-gray-12 font-medium text-xs">
                  {formatCpu(deployment.cpuMillicores)}
                </div>
              </InfoChip>
              <InfoChip icon={Grid}>
                <div className="text-gray-12 font-medium text-xs">
                  {formatMemory(deployment.memoryMib)}
                </div>
              </InfoChip>
              <DisabledWrapper tooltipContent="Storage metrics coming soon">
                <InfoChip icon={Harddrive}>
                  <div className="text-grayA-10 text-xs">
                    <span className="text-gray-12 font-medium">—</span> GB
                  </div>
                </InfoChip>
              </DisabledWrapper>
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
