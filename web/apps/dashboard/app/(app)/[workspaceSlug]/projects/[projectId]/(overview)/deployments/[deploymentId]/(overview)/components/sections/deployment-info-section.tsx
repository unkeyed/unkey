"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { Bolt, Cloud, Grid, Harddrive, LayoutRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useParams } from "next/navigation";
import { ActiveDeploymentCard } from "../../../../../../components/active-deployment-card";
import { DeploymentStatusBadge } from "../../../../../../components/deployment-status-badge";
import { DisabledWrapper } from "../../../../../../components/disabled-wrapper";
import { InfoChip } from "../../../../../../components/info-chip";
import { Section, SectionHeader } from "../../../../../../components/section";
import { useProject } from "../../../../../layout-provider";

export function DeploymentInfoSection() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;

  const { collections, setIsDetailsOpen, isDetailsOpen } = useProject();
  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );
  const deploymentStatus = deployment.data.at(0)?.status;

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
              className="flex gap-1.5 items-center"
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
            <div className="gap-1 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
              <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                <img src={"/images/flags/us.svg"} alt="us-flag" className="size-4" />
              </div>
              <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                <img src={"/images/flags/de.svg"} alt="de-flag" className="size-4" />
              </div>
              <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                <img src={"/images/flags/in.svg"} alt="in-flag" className="size-4" />
              </div>
            </div>
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
        statusBadge={
          <DeploymentStatusBadge
            status={deploymentStatus}
            className="text-successA-11 font-medium"
          />
        }
      />
    </Section>
  );
}
