"use client";

import { cn } from "@/lib/utils";
import { trpc } from "@/lib/trpc/client";
import { Check, CloudUp, Earth, Hammer2, LayerFront, TriangleWarning2 } from "@unkey/icons";
import { Badge, Loading, SettingCard, SettingCardGroup } from "@unkey/ui";
import ms from "ms";
import { useEffect, useState } from "react";
import { useDeployment } from "../../../layout-provider";
import { DeploymentBuildStepsTable } from "../table/deployment-build-steps-table";
import { DeploymentInfoSection } from "./deployment-info-section";

export function DeploymentProgressSection() {
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

  const [now, setNow] = useState(0);
  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now()), 500);
    return () => {
      clearInterval(interval);
    };
  }, []);
  const { building, deploying, network, queued } = steps.data ?? {};

  return (
    <>
      <DeploymentInfoSection />
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
                ? (network.error ?? "Assigned all domains")
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
    </>
  );
}

type DeploymentStepProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  duration?: number;
  status: "pending" | "started" | "completed" | "error";
  expandable?: React.ReactNode;
  defaultExpanded?: boolean;
};

function DeploymentStep({
  icon,
  title,
  description,
  duration,
  status,
  expandable,
  defaultExpanded,
}: DeploymentStepProps) {
  const showGlow = status === "started"
  return (
    <SettingCard
      icon={
        <div className="relative w-full h-full">
          <div
            className={cn(
              "absolute inset-[-2px] rounded-[8px] blur-[10px] transition-opacity duration-300",
              "bg-gradient-to-l from-feature-8 to-info-9",
              showGlow ? "opacity-50" : "opacity-0",
            )}
          />
          <div className={cn("w-full h-full rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20", showGlow && "relative dark:bg-white dark:text-black bg-black text-white shadow-md shadow-black/40")}>
            {icon}
          </div>
        </div>
      }
      title={
        <div className="flex items-center gap-2">
          <span>{title}</span>
          <Badge
            variant="success"
            size="sm"
            className={cn(
              "transition-all duration-300 font-normal text-[11px] rounded-md h-[18px]",
              status === "completed" ? "opacity-100 scale-100" : "opacity-0 scale-95",
            )}
          >
            Complete
          </Badge>
        </div>
      }
      className="relative"
      description={description}
      expandable={expandable}
      defaultExpanded={defaultExpanded}
      contentWidth="w-fit"
    >
      <div className="flex items-center gap-4 justify-end w-full absolute right-14">
        <span className="text-gray-10 text-xs">{duration ? ms(duration) : null}</span>
        {status === "completed" ? (
          <Check iconSize="md-regular" className="text-success-11" />
        ) : status === "started" ? (
          <Loading className="size-4" />
        ) : status === "error" ? (
          <TriangleWarning2 className="text-error-11" />
        ) : null}
      </div>
    </SettingCard>
  );
}
