"use client";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo } from "react";
import { DeploymentDomainsCard } from "../../../components/deployment-domains-card";
import type { DeploymentStatus } from "../../../components/deployment-status-badge";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentInfo } from "./(deployment-progress)/deployment-info";
import { DeploymentProgress, type StepsData } from "./(deployment-progress)/deployment-progress";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
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

const DEPLOYMENT_STATUSES: ReadonlySet<string> = new Set<DeploymentStatus>([
  "pending",
  "building",
  "deploying",
  "network",
  "ready",
  "failed",
]);

function isDeploymentStatus(value: string): value is DeploymentStatus {
  return DEPLOYMENT_STATUSES.has(value);
}

function deriveStatusFromSteps(steps: StepsData | undefined, fallback: string): DeploymentStatus {
  if (!steps) {
    return isDeploymentStatus(fallback) ? fallback : "pending";
  }

  const { queued, building, deploying, network } = steps;

  if ([queued, building, deploying, network].some((s) => s?.error)) {
    return "failed";
  }
  if (network?.completed) {
    return "ready";
  }
  if (network && !network.endedAt) {
    return "network";
  }
  if (deploying && !deploying.endedAt) {
    return "deploying";
  }
  if (building && !building.endedAt) {
    return "building";
  }
  if (queued && !queued.endedAt) {
    return "pending";
  }

  return isDeploymentStatus(fallback) ? fallback : "pending";
}
