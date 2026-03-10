"use client";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo } from "react";
import { DeploymentDomainsCard } from "../../../components/deployment-domains-card";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentInfo } from "./(deployment-progress)/deployment-info";
import { DeploymentProgress } from "./(deployment-progress)/deployment-progress";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { deriveStatusFromSteps } from "./deployment-utils";
import { useDeployment } from "./layout-provider";

export default function DeploymentOverview() {
  const { deployment } = useDeployment();
  const { refetchDomains } = useProjectData();

  const ready = deployment.status === "ready";

  const stepsQuery = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    { refetchInterval: ready ? false : 1_000, refetchOnWindowFocus: false },
  );

  const derivedStatus = useMemo(
    () => deriveStatusFromSteps(stepsQuery.data, deployment.status),
    [stepsQuery.data, deployment.status],
  );

  useEffect(() => {
    if (ready) {
      stepsQuery.refetch();
      refetchDomains();
    }
  }, [ready, refetchDomains, stepsQuery.refetch]);

  return (
    <ProjectContentWrapper centered>
      <DeploymentInfo statusOverride={derivedStatus} />
      {ready ? (
        <div key="ready" className="flex flex-col gap-5 animate-fade-slide-in">
          <DeploymentDomainsCard />
          <DeploymentNetworkSection />
        </div>
      ) : (
        <div key="progress" className="animate-fade-slide-in">
          <DeploymentProgress stepsData={stepsQuery.data} />
        </div>
      )}
    </ProjectContentWrapper>
  );
}
