"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { CloudUp, Earth, Hammer2, LayerFront, Pulse, Sparkle3 } from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { DeploymentDomainsCard } from "../../../../components/deployment-domains-card";
import { useProjectData } from "../../../data-provider";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";
import { useDeployment } from "../layout-provider";
import { DeploymentBuildStepsTable } from "./build-steps-table/deployment-build-steps-table";
import { DeploymentStep } from "./deployment-step";
import { resolveDeploymentStep } from "./deployment-step-resolution";

type RouterOutputs = inferRouterOutputs<Router>;
export type StepsData = RouterOutputs["deploy"]["deployment"]["steps"];

export function DeploymentProgress({ stepsData }: { stepsData?: StepsData }) {
  const { deployment } = useDeployment();
  const router = useRouter();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;
  const isFailed = deployment.status === "failed";

  const buildSteps = trpc.deploy.deployment.buildSteps.useQuery(
    {
      deploymentId: deployment.id,
      includeStepLogs: true,
    },
    {
      refetchInterval: 1_000,
    },
  );

  const { getDomainsForDeployment, projectId } = useProjectData();

  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (isFailed) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), 500);
    return () => {
      clearInterval(interval);
    };
  }, [isFailed]);

  const { building, deploying, network, queued, starting, finalizing } = stepsData ?? {};

  const queuedImplicitlyComplete =
    !queued && Boolean(starting ?? building ?? deploying ?? network ?? finalizing);

  const [redeployOpen, setRedeployOpen] = useState(false);
  const domainsForDeployment = getDomainsForDeployment(deployment.id);

  // Latch true once we observe the build actively in progress; stays true after it completes
  const hasFreshBuild = useRef(false);
  if (building && !building.endedAt) {
    hasFreshBuild.current = true;
  }
  const isPrebuilt = !hasFreshBuild.current && !building?.error;

  const queuedStep = resolveDeploymentStep({
    step: queued,
    now,
    isFailed,
    skippable: false,
    implicitlyComplete: queuedImplicitlyComplete,
    completedMessage: "Deployment has queued",
    inProgressMessage: "Deployment is queued",
    waitingMessage: "Waiting to queue",
  });

  const startingStep = resolveDeploymentStep({
    step: starting,
    now,
    isFailed,
    skippable: false,
    completedMessage: "Deployment has started",
    inProgressMessage: "Deployment has started",
    waitingMessage: "Preparing deployment for building",
  });

  const deployingStep = resolveDeploymentStep({
    step: deploying,
    now,
    isFailed,
    skippable: true,
    completedMessage: "Deployed to all machines",
    inProgressMessage: "Deploying to all machines",
    waitingMessage: "Waiting for image build",
  });

  const networkStep = resolveDeploymentStep({
    step: network,
    now,
    isFailed,
    skippable: true,
    completedMessage: `Domains assigned · ${domainsForDeployment.length} records`,
    inProgressMessage: "Assigning domains",
    waitingMessage: "Waiting for containers to deploy",
  });

  const finalizingStep = resolveDeploymentStep({
    step: finalizing,
    now,
    isFailed,
    skippable: true,
    completedMessage: "Deployment has finished",
    inProgressMessage: "Finalizing deployment",
    waitingMessage: "Waiting for domains",
  });

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
          {...queuedStep}
        />
        <DeploymentStep
          icon={<Pulse iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Starting"
          {...startingStep}
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
              : deploying
                ? "Image was prebuilt"
                : "Waiting for deployment to start"
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
          {...deployingStep}
        />
        <DeploymentStep
          icon={<Earth iconSize="sm-medium" className="size-[18px]" />}
          title="Assigning Domains"
          {...networkStep}
        />
        <DeploymentStep
          icon={<Sparkle3 iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment finalizing"
          {...finalizingStep}
        />
      </SettingCardGroup>
      {isFailed && (
        <div className="flex flex-col gap-3 animate-fade-slide-in">
          <div className="border border-errorA-4 bg-errorA-2 rounded-[14px] p-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex flex-col gap-0.5">
                <span className="text-sm font-medium text-error-11">Deployment failed</span>
                <span className="text-xs text-gray-11">
                  {[queued, starting, building, deploying, network, finalizing].find(
                    (s) => s?.error,
                  )?.error ?? "Deployment failed"}
                </span>
              </div>
            </div>
            <Button
              variant="primary"
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
