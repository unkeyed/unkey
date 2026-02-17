"use client";
import { useEffect } from "react";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentDomainsSection } from "./(overview)/components/sections/deployment-domains-section";
import { DeploymentInfoSection } from "./(overview)/components/sections/deployment-info-section";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { DeploymentProgressSection } from "./(overview)/components/sections/deployment-progress-section";
import { useDeployment } from "./layout-provider";

export default function DeploymentOverview() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById, refetchDomains } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  const ready = deployment?.status === "ready";

  useEffect(() => {
    if (ready) {
      refetchDomains();
    }
  }, [ready, refetchDomains]);

  if (!ready) {
    return (
      <ProjectContentWrapper centered>
        <DeploymentProgressSection />
      </ProjectContentWrapper>
    );
  }

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfoSection />

      <DeploymentDomainsSection />
      <DeploymentNetworkSection />
    </ProjectContentWrapper>
  );
}
