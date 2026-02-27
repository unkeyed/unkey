"use client";

import { trpc } from "@/lib/trpc/client";
import { CloudUp, Earth, Hammer2, LayerFront } from "@unkey/icons";
import { SettingCardGroup } from "@unkey/ui";
import { useEffect, useState } from "react";
import { DeploymentDomainsCard } from "../../../../components/deployment-domains-card";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";
import { DeploymentBuildStepsTable } from "./build-steps-table/deployment-build-steps-table";
import { DeploymentStep } from "./deployment-step";

export function DeploymentProgress() {
  const { deployment } = useDeployment();
  const steps = trpc.deploy.deployment.steps.useQuery(
    {
      deploymentId: deployment.id,
    },
    {
      refetchInterval: 1_000,
    },
  );

  const buildSteps = trpc.deploy.deployment.buildSteps.useQuery(
    {
      deploymentId: deployment.id,
      includeStepLogs: true,
    },
    {
      refetchInterval: 1000,
    },
  );

  const { getDomainsForDeployment } = useProjectData();

  const [now, setNow] = useState(0);
  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now()), 500);
    return () => {
      clearInterval(interval);
    };
  }, []);
  const { building, deploying, network, queued } = steps.data ?? {};

  const domainsForDeployment = getDomainsForDeployment(deployment.id);

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<Hammer2 iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Queued"
          description={
            queued
              ? queued.endedAt
                ? (queued.error ?? "Deployment has started")
                : "Deployment is queued"
              : "Waiting deployment to start"
          }
          duration={queued ? (queued.endedAt ?? now) - queued.startedAt : undefined}
          status={
            queued?.error
              ? "error"
              : queued?.completed
                ? "completed"
                : queued
                  ? "started"
                  : "pending"
          }
        />
        <DeploymentStep
          icon={<LayerFront iconSize="sm-medium" className="size-[18px]" />}
          title="Building Image"
          description={
            building
              ? building.endedAt
                ? (building.error ?? "Build Complete")
                : (buildSteps.data?.steps.at(-1)?.name ?? "Building...")
              : "Waiting for build runner"
          }
          duration={building ? (building.endedAt ?? now) - building.startedAt : undefined}
          status={
            building?.error
              ? "error"
              : building?.completed
                ? "completed"
                : building
                  ? "started"
                  : "pending"
          }
          expandable={
            <div className="bg-grayA-2">
              <DeploymentBuildStepsTable steps={buildSteps.data?.steps ?? []} />
            </div>
          }
          defaultExpanded
        />
        <DeploymentStep
          icon={<CloudUp iconSize="sm-medium" className="size-[18px]" />}
          title="Deploying Containers"
          description={
            deploying
              ? deploying.endedAt
                ? (deploying.error ?? "Deployed to all machines")
                : "Deploying to all machines"
              : "Waiting for build"
          }
          duration={deploying ? (deploying.endedAt ?? now) - deploying.startedAt : undefined}
          status={
            deploying?.error
              ? "error"
              : deploying?.completed
                ? "completed"
                : deploying
                  ? "started"
                  : "pending"
          }
        />
        <DeploymentStep
          icon={<Earth iconSize="sm-medium" className="size-[18px]" />}
          title="Assigning Domains"
          description={
            network
              ? network.endedAt
                ? (network.error ?? `Domains assigned · ${domainsForDeployment.length} records`)
                : "Assigning domains"
              : "Waiting for deployments"
          }
          duration={network ? (network.endedAt ?? now) - network.startedAt : undefined}
          status={
            network?.error
              ? "error"
              : network?.completed
                ? "completed"
                : network
                  ? "started"
                  : "pending"
          }
        />
      </SettingCardGroup>
      {network?.completed && (
        <div className="animate-fade-slide-in">
          <DeploymentDomainsCard glow />
        </div>
      )}
    </div>
  );
}
