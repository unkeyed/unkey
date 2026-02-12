"use client";
import { useEffect } from "react";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentDomainsSection } from "./(overview)/components/sections/deployment-domains-section";
import { DeploymentInfoSection } from "./(overview)/components/sections/deployment-info-section";
import { DeploymentLogsSection } from "./(overview)/components/sections/deployment-logs-section";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { useDeployment } from "./layout-provider";

export default function DeploymentOverview() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById, refetchDomains } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  useEffect(() => {
    if (deployment?.status === "ready") {
      refetchDomains();
    }
  }, [deployment, refetchDomains]);

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfoSection />

      {deployment?.status === "ready" ? (
        <>
          <DeploymentDomainsSection />
          <DeploymentNetworkSection />
        </>
      ) : (
        <DeploymentLogsSection />
      )}
    </ProjectContentWrapper>
  );
}
