"use client";

import { cn } from "@/lib/utils";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, CloudUp, Cube, Earth, Hammer2, LayerFront, Link4 } from "@unkey/icons";
import { Button, CopyButton, Popover, PopoverContent, PopoverTrigger, SettingCard, SettingCardGroup } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useProjectData } from "../../../data-provider";
import { SettingsGroup } from "../../../settings/components/shared/settings-group";
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

  const { getDomainsForDeployment, isDomainsLoading, project } = useProjectData();

  const [now, setNow] = useState(0);
  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now()), 500);
    return () => {
      clearInterval(interval);
    };
  }, []);
  const { building, deploying, network, queued } = steps.data ?? {};

  const domainsForDeployment = getDomainsForDeployment(deployment.id);
  const sortedDomains = [...domainsForDeployment].sort((a, b) =>
    a.fullyQualifiedDomainName.localeCompare(b.fullyQualifiedDomainName),
  );
  const environmentDomain = sortedDomains.find((d) => d.sticky === "environment");
  const primaryDomain = environmentDomain ?? sortedDomains[0];
  const additionalDomains = primaryDomain
    ? sortedDomains.filter((d) => d.id !== primaryDomain.id)
    : [];

  const showDomains = network?.completed && (isDomainsLoading || sortedDomains.length > 0);

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
                ? (network.error ?? `Domains assigned · ${sortedDomains.length} records`)
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
      {showDomains && (
        <SettingsGroup
          icon={<Earth iconSize="md-medium" />}
          title={<span className="font-medium text-gray-12 text-[13px] leading-4">Domains</span>}
          hideChevron
        >
          <SettingCardGroup>
            {isDomainsLoading ? (
              <SettingCard
                icon={
                  <div className="w-full h-full rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4 dark:shadow-none shadow-sm shadow-grayA-8/20">
                    <Earth iconSize="sm-medium" className="size-[18px]" />
                  </div>
                }
                title={<div className="h-4 w-36 bg-grayA-3 rounded animate-pulse" />}
                description="Loading domains..."
              />
            ) : (
              <SettingCard
                icon={
                  <div className="relative w-full h-full">
                    <div
                      className={cn(
                        "absolute inset-[-4px] rounded-[10px] blur-[14px]",
                        "bg-gradient-to-l from-feature-8 to-info-9",
                        "animate-pulse opacity-20"
                      )}
                    />
                    <div className={cn("w-full h-full rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4 dark:shadow-none relative dark:bg-white dark:text-black bg-black text-white")}>
                      <Cube iconSize="md-medium" className="size-[18px]" />
                    </div>
                  </div>
                }
                title={project?.name}
                description={<div className="flex items-center justify-center gap-2 ">
                  <span className="text-gray-10 text-xs">{primaryDomain.fullyQualifiedDomainName}</span>
                  <div className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-xs leading-[18px] font-mono tabular-nums hover:bg-grayA-4 transition-colors">
                    +{additionalDomains.length}
                  </div>
                </div>}
                contentWidth="w-fit"
              >
                <div className="flex items-center gap-2">
                  {additionalDomains.length > 0 && (
                    <Popover>
                      <PopoverTrigger asChild>
                        <Button className="text-gray-12 font-medium bg-grayA-2" variant="outline">
                          Copy URL
                          <ChevronDown className="text-gray-9 !size-3" iconSize="sm-regular" />
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent side="bottom" align="end" className="p-0 max-w-[300px]">
                        {additionalDomains.map((d) => (
                          <div key={d.id} className="flex items-center justify-left w-full h-10 border-b border-gray-4 px-3 py-[14px] gap-2">
                            <Link4 className="text-gray-9 !size-3 shrink-0" iconSize="sm-regular" />
                            <a
                              href={`https://${d.fullyQualifiedDomainName}`}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="transition-all hover:underline decoration-dashed underline-offset-2 max-w-[250px] truncate font-medium text-xs"
                            >
                              {d.fullyQualifiedDomainName}
                            </a>
                            <CopyButton
                              value={`https://${d.fullyQualifiedDomainName}`}
                              variant="ghost"
                              toastMessage={d.fullyQualifiedDomainName}
                            />
                          </div>
                        ))}
                      </PopoverContent>
                    </Popover>
                  )}
                </div>
              </SettingCard>
            )}
          </SettingCardGroup>
        </SettingsGroup>
      )}
    </div>
  );
}
