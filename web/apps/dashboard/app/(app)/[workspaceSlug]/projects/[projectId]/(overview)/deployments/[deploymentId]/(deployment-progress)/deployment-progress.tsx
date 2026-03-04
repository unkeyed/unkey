"use client";

import { trpc } from "@/lib/trpc/client";
import { CloudUp, Earth, Hammer2, LayerFront } from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { DeploymentDomainsCard } from "../../../../components/deployment-domains-card";
import { useProjectData } from "../../../data-provider";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";
import { useDeployment } from "../layout-provider";
import { DeploymentBuildStepsTable } from "./build-steps-table/deployment-build-steps-table";
import { DeploymentStep } from "./deployment-step";

export function DeploymentProgress() {
  const { deployment } = useDeployment();
  const router = useRouter();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;
  const projectId = params.projectId as string;
  const isFailed = deployment.status === "failed";

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
      refetchInterval: 1_000,
    },
  );

  const { getDomainsForDeployment } = useProjectData();

  const [now, setNow] = useState(0);
  useEffect(() => {
    if (isFailed) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), 500);
    return () => {
      clearInterval(interval);
    };
  }, [isFailed]);

  const { building, deploying, network, queued } = steps.data ?? {};

  const [redeployOpen, setRedeployOpen] = useState(false);
  const domainsForDeployment = getDomainsForDeployment(deployment.id);

  // Latch true once we observe the build actively in progress; stays true after it completes
  const hasFreshBuild = useRef(false);
  if (building && !building.endedAt) {
    hasFreshBuild.current = true;
  }
  const isPrebuilt = !hasFreshBuild.current && !building?.error;

  useEffect(() => {
    if (network?.completed) {
      router.push(`/${workspaceSlug}/projects/${projectId}/deployments/${deployment.id}`);
    }
  }, [network?.completed, router, workspaceSlug, projectId, deployment.id]);

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<LayerFront iconSize="sm-medium" className="size-[18px]" />}
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
          key={isPrebuilt ? "prebuilt" : "building"}
          icon={<Hammer2 iconSize="sm-medium" className="size-[18px]" />}
          title="Building Image"
          description={
            building
              ? building.endedAt
                ? (building.error ??
                  (hasFreshBuild.current ? "Build Complete" : "Image was prebuilt"))
                : (buildSteps.data?.steps.at(-1)?.name ?? "Building...")
              : "Image was prebuilt"
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
            isPrebuilt ? null : (
              <div className="bg-grayA-2">
                <DeploymentBuildStepsTable steps={buildSteps.data?.steps ?? []} />
              </div>
            )
          }
          defaultExpanded={!isPrebuilt}
        />
        <DeploymentStep
          icon={<CloudUp iconSize="sm-medium" className="size-[18px]" />}
          title="Deploying Containers"
          description={
            deploying
              ? deploying.endedAt
                ? (deploying.error ?? "Deployed to all machines")
                : "Deploying to all machines"
              : isFailed
                ? "Skipped"
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
                  : isFailed
                    ? "skipped"
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
              : isFailed
                ? "Skipped"
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
                  : isFailed
                    ? "skipped"
                    : "pending"
          }
        />
      </SettingCardGroup>
      {isFailed && (
        <div className="flex flex-col gap-3 animate-fade-slide-in">
          <div className="border border-errorA-4 bg-errorA-2 rounded-[14px] p-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex flex-col gap-0.5">
                <span className="text-sm font-medium text-error-11">Deployment failed</span>
                <span className="text-xs text-gray-11">
                  {[queued, building, deploying, network].find((s) => s?.error)?.error ??
                    "Deployment failed"}
                </span>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setRedeployOpen(true)}
              className="px-3"
            >
              Redeploy
            </Button>
          </div>
          <RedeployDialog
            isOpen={redeployOpen}
            onClose={() => setRedeployOpen(false)}
            selectedDeployment={deployment}
          />
        </div>
      )}
      {network?.completed && (
        <div className="animate-fade-slide-in">
          <DeploymentDomainsCard glow />
        </div>
      )}
    </div>
  );
}
