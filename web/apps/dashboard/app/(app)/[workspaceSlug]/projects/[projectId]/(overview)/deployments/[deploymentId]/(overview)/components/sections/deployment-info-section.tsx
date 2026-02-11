"use client";

import { Bolt, Cloud, Grid, Harddrive, LayoutRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { ActiveDeploymentCard } from "../../../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../../../components/deployment-status-badge";
import { DisabledWrapper } from "../../../../../../components/disabled-wrapper";
import { InfoChip } from "../../../../../../components/info-chip";
import { RegionFlags } from "../../../../../../components/region-flags";
import { Section, SectionHeader } from "../../../../../../components/section";
import { useProjectData } from "../../../../../data-provider";
import { useProjectLayout } from "../../../../../layout-provider";
import { useDeployment } from "../../../layout-provider";

export function DeploymentInfoSection() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById } = useProjectData();
  const { setIsDetailsOpen, isDetailsOpen } = useProjectLayout();

  const deployment = getDeploymentById(deploymentId);
  const deploymentStatus = deployment?.status;

  return (
    <Section>
      <SectionHeader
        icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
        title="Deployment"
      />
      <ActiveDeploymentCard
        deploymentId={deploymentId}
        trailingContent={
          <div className="flex gap-1.5 items-center">
            <DisabledWrapper
              tooltipContent="Resource metrics coming soon"
              className="2xl:flex gap-1.5 items-center hidden"
            >
              <InfoChip icon={Bolt}>
                <div className="text-grayA-10 text-xs">
                  <span className="text-gray-12 font-medium">—</span> vCPUs
                </div>
              </InfoChip>
              <InfoChip icon={Grid}>
                <div className="text-grayA-10 text-xs">
                  <span className="text-gray-12 font-medium">—</span> GiB
                </div>
              </InfoChip>
              <InfoChip icon={Harddrive}>
                <div className="text-grayA-10 text-xs">
                  <span className="text-gray-12 font-medium">—</span> GB
                </div>
              </InfoChip>
            </DisabledWrapper>
            <RegionFlags instances={deployment?.instances ?? []} />
            <InfoTooltip asChild content="Show deployment details">
              <Button
                variant="ghost"
                className="[&_svg]:size-3 size-3 rounded-sm"
                size="icon"
                onClick={() => setIsDetailsOpen(!isDetailsOpen)}
              >
                <LayoutRight iconSize="sm-regular" className="text-gray-10" />
              </Button>
            </InfoTooltip>
          </div>
        }
        statusBadge={<DeploymentStatusBadge status={deploymentStatus} />}
      />
    </Section>
  );
}
