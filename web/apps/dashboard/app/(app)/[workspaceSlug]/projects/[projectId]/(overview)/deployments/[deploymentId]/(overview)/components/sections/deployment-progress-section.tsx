"use client";

import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Check, ChevronRight, TriangleWarning2, Ufo } from "@unkey/icons";
import { Badge, Loading } from "@unkey/ui";
import ms from "ms";
import { useEffect, useState } from "react";
import { Card } from "../../../../../components/card";
import { useDeployment } from "../../../layout-provider";
import { DeploymentBuildStepsTable } from "../table/deployment-build-steps-table";
import { DeploymentInfoSection } from "./deployment-info-section";

export function DeploymentProgressSection() {
  const { deploymentId } = useDeployment();
  const steps = trpc.deploy.deployment.steps.useQuery(
    {
      deploymentId,
    },
    {
      refetchInterval: 1_000,
    },
  );

  const buildSteps = trpc.deploy.deployment.buildSteps.useQuery(
    {
      deploymentId,
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

      <Card className="rounded-[14px] divide-y divide-gray-4 flex justify-between flex-col overflow-hidden border-gray-4">
        <Step
          icon={<Ufo />}
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
          defaultExpanded={true}
        />
        <Step
          icon={<Ufo />}
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
          expanded={<DeploymentBuildStepsTable steps={buildSteps.data?.steps ?? []} />}
          defaultExpanded={true}
        />
        <Step
          icon={<Ufo />}
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
        <Step
          icon={<Ufo />}
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
      </Card>
    </>
  );
}

type StepProps = {
  icon: React.ReactNode;
  title: string;
  description: string;
  duration?: number;
  status: "pending" | "started" | "completed" | "error";
  defaultExpanded?: boolean;
  expanded?: React.ReactNode;
};

const Step: React.FC<StepProps> = ({
  icon,
  title,
  description,
  duration,
  status,
  defaultExpanded,
  expanded,
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <div className="py-4 flex justify-between flex-col overflow-hidden">
      <div className="px-4 flex w-full justify-between items-center ">
        <div className="flex gap-5 items-center">
          {icon}
          <div className="flex flex-col gap-1">
            <div className=" gap-2 flex items-center">
              <span className="text-gray-12 font-medium text-sm">{title}</span>
              {status === "completed" ? (
                <Badge variant="success" size="sm">
                  Complete
                </Badge>
              ) : null}
            </div>
            <p className="text-gray-10 text-xs">{description}</p>
          </div>
        </div>
        <div className="items-center flex gap-2">
          <div className="flex gap-2 items-center">
            <span className="text-gray-10 text-xs">{duration ? ms(duration) : null}</span>
            {status === "completed" ? (
              <Check iconSize="md-thin" className="text-success-11" />
            ) : status === "started" ? (
              <Loading />
            ) : status === "error" ? (
              <TriangleWarning2 className="text-error-11" />
            ) : null}
            <div className="w-4">
              {expanded ? (
                <button onClick={() => setIsExpanded(!isExpanded)} type="button">
                  <ChevronRight
                    iconSize="md-thin"
                    className={cn("text-gray-10", { "rotate-90": isExpanded })}
                  />
                </button>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      <div>{isExpanded ? expanded : null}</div>
    </div>
  );
};
