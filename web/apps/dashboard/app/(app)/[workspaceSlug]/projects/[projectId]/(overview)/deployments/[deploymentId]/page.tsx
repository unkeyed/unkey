"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { Cube } from "@unkey/icons";
import { Card } from "@unkey/ui";
import { useParams } from "next/navigation";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { Section, SectionHeader } from "../../../components/section";
import { useProject } from "../../layout-provider";
import { DeploymentDomainsSection } from "./(overview)/components/sections/deployment-domains-section";
import { DeploymentInfoSection } from "./(overview)/components/sections/deployment-info-section";
import { DeploymentLogsSection } from "./(overview)/components/sections/deployment-logs-section";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { DeploymentBuildStepsTable } from "./(overview)/components/table/deployment-build-steps-table";

export default function DeploymentOverview() {
  const { collections } = useProject();

  const params = useParams();
  const deploymentId = params?.deploymentId as string;

  const deployments = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );

  const isReady = deployments.data?.at(0)?.status === "ready";

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfoSection />
      {isReady ? <DeploymentDomainsSection /> : null}
      {isReady ? <DeploymentNetworkSection /> : null}
      {isReady ? (
        <DeploymentLogsSection />
      ) : (
        <Section>
          <SectionHeader
            icon={<Cube iconSize="md-regular" className="text-gray-9" />}
            title="Build Logs"
          />
          <Card className="rounded-[14px] overflow-hidden border-gray-4 flex flex-col h-full">
            <DeploymentBuildStepsTable />
          </Card>
        </Section>
      )}
    </ProjectContentWrapper>
  );
}
