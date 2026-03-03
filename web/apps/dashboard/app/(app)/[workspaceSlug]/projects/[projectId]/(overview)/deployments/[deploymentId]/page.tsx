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
      {ready ? (
        <div key="ready" className="flex flex-col gap-5 animate-fade-slide-in">
          <DeploymentDomainsCard />
          <DeploymentNetworkSection />
        </div>
      ) : (
        <div key="progress" className="animate-fade-slide-in">
          <DeploymentProgress />
        </div>
      )}
    </ProjectContentWrapper>
  );
}
