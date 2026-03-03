"use client";
import { useEffect } from "react";
import { DeploymentDomainsCard } from "../../../components/deployment-domains-card";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentInfo } from "./(deployment-progress)/deployment-info";
import { DeploymentProgress } from "./(deployment-progress)/deployment-progress";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { useDeployment } from "./layout-provider";

export default function DeploymentOverview() {
  const { deployment } = useDeployment();
  const { refetchDomains } = useProjectData();

  const ready = deployment.status === "ready";

  useEffect(() => {
    if (ready) {
      refetchDomains();
    }
  }, [ready, refetchDomains]);

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfo />
      <DeploymentProgress />

    </ProjectContentWrapper>
  );
}
