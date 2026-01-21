
"use client";

import { ReactNode } from "react";
import { ProjectContentWrapper } from "../components/project-content-wrapper";
import { Bolt, Cloud, Grid, Harddrive, LayoutRight } from "@unkey/icons";
import { ActiveDeploymentCard } from "../components/active-deployment-card";
import { DeploymentStatusBadge } from "../components/deployment-status-badge";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProject } from "../(overview)/layout-provider";
import { InfoChip } from "../components/info-chip";
import { Button, InfoTooltip } from "@unkey/ui";


const DEPLOYMENT_ID = "d_5VmWaBhBEn5jmAcZ"

export default function DeploymentOverview() {
  const { collections, setIsDetailsOpen, isDetailsOpen } = useProject();
  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, DEPLOYMENT_ID)),
    [DEPLOYMENT_ID],
  );
  const deploymentStatus = deployment.data.at(0)?.status;

  return (
    <ProjectContentWrapper centered>
      <Section>
        <SectionHeader
          icon={<Cloud iconSize="md-regular" className="text-gray-9" />}
          title="Deployment"
        />
        <ActiveDeploymentCard deploymentId={DEPLOYMENT_ID}
          trailingContent={<div className="flex gap-1.5 items-center">
            <InfoChip icon={Bolt}>
              <div className="text-grayA-10 text-xs">
                <span className="text-gray-12 font-medium">4</span> vCPUs
              </div>
            </InfoChip>
            <InfoChip icon={Grid}>
              <div className="text-grayA-10 text-xs">
                <span className="text-gray-12 font-medium">4</span> GiB
              </div>
            </InfoChip>
            <InfoChip icon={Harddrive}>
              <div className="text-grayA-10 text-xs">
                <span className="text-gray-12 font-medium">20</span> GB
              </div>
            </InfoChip>
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
              <Button variant="ghost" className="[&_svg]:size-3 size-3 rounded-sm" size="icon" onClick={() => setIsDetailsOpen(!isDetailsOpen)}>
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
    </ProjectContentWrapper >
  );
}

function SectionHeader({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-2.5 py-1.5 px-2">
      {icon}
      <div className="text-accent-12 font-medium text-[13px] leading-4">{title}</div>
    </div>
  );
}

function Section({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-1">{children}</div>;
}



