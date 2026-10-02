"use client";

import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import {
  IconCloudUploadOutline18,
  IconEarthOutline18,
  IconHammer2Outline18,
  IconLayerFrontOutline18,
  IconSparkle3Outline18,
} from "@unkey/icons";
import { P, match } from "@unkey/match";
import { SettingCardGroup } from "@unkey/ui";
import { useParams, usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { DeploymentDomainsCard } from "../../../../components/deployment-domains-card";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";
import { DeploymentBuildLogs } from "./build-logs/deployment-build-logs";
import { DeploymentContainerLogsTable } from "./container-logs-table/deployment-container-logs-table";
import { DeploymentStep } from "./deployment-step";
import { resolveDeploymentStep } from "./deployment-step-resolution";

type RouterOutputs = inferRouterOutputs<Router>;
export type StepsData = RouterOutputs["deploy"]["deployment"]["steps"];

export function DeploymentProgress({ stepsData }: { stepsData?: StepsData }) {
  const { deployment } = useDeployment();
  const router = useRouter();
  const pathname = usePathname();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;
  const isFailed = deployment.status === "failed";

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

  const { building, deploying, network, queued, finalizing } = stepsData ?? {};

  const deploymentRuntimeLogs = trpc.deploy.deployment.runtimeLogs.useQuery(
    { deploymentId: deployment.id, limit: 50 },
    {
      refetchInterval:
        isDeploymentInFlight(deployment.status) && deploying && !deploying.endedAt ? 2_000 : false,
    },
  );

  const queuedImplicitlyComplete =
    !queued && Boolean(building ?? deploying ?? network ?? finalizing);

  const domainsForDeployment = getDomainsForDeployment(deployment.id);

  // Latch true once we observe the build actively in progress; stays true after it completes
  const hasFreshBuild = useRef(false);
  if (building && !building.endedAt) {
    hasFreshBuild.current = true;
  }
  const isPrebuilt = !hasFreshBuild.current && !building?.error;

  const [buildErrorFocusTick, setBuildErrorFocusTick] = useState(0);
  // The steps load after the first render, so the card follows isPrebuilt
  // until the user opens or closes it
  const [buildExpandedChoice, setBuildExpandedChoice] = useState<boolean>();
  const buildExpanded = buildExpandedChoice ?? !isPrebuilt;
  const buildError = building?.error;

  const revealBuildError = () => {
    setBuildExpandedChoice(true);
    setBuildErrorFocusTick((tick) => tick + 1);
  };

  // biome-ignore lint/correctness/useExhaustiveDependencies: revealBuildError is new on every render; buildError and pathname are the triggers
  useEffect(() => {
    if (!buildError || isPrebuilt) {
      return;
    }
    revealBuildError();
  }, [buildError, isPrebuilt, pathname]);

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
      router.push(
        routes.projects.apps.deployment({
          workspaceSlug,
          projectId,
          appId: deployment.appId,
          deploymentId: deployment.id,
        }),
      );
    }
  }, [network?.completed, router, workspaceSlug, projectId, deployment.appId, deployment.id]);

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<IconLayerFrontOutline18 />}
          title="Deployment Queued"
          {...queuedStep}
        />
        <DeploymentStep
          key={isPrebuilt ? "prebuilt" : "building"}
          icon={<IconHammer2Outline18 />}
          title="Building Image"
          truncateDescription={!building?.error}
          description={match(building)
            .when(
              (b) => Boolean(b?.error),
              (b) => <BuildErrorDescription error={b?.error ?? ""} onViewLogs={revealBuildError} />,
            )
            .when(
              (b) => Boolean(b?.endedAt),
              () => (hasFreshBuild.current ? "Build Complete" : "Image was prebuilt"),
            )
            .with(P.nonNullable, () => "Building...")
            .otherwise(() =>
              deploying ? "Image was prebuilt" : "Waiting for deployment to start",
            )}
          duration={building ? (building.endedAt ?? now) - building.startedAt : undefined}
          status={match(building)
            .when(
              (b) => Boolean(b?.error),
              () => "error" as const,
            )
            .when(
              (b) => Boolean(b?.completed),
              () => "completed" as const,
            )
            .with(P.nonNullable, () => "started" as const)
            .otherwise(() => "pending" as const)}
          expandable={
            isPrebuilt ? null : (
              <div className="bg-grayA-2">
                <DeploymentBuildLogs focusErrorTick={buildErrorFocusTick} isOpen={buildExpanded} />
              </div>
            )
          }
          expanded={buildExpanded}
          onExpandedChange={setBuildExpandedChoice}
        />
        <DeploymentStep
          key={deploying ? "deploying-active" : "deploying-pending"}
          icon={<IconCloudUploadOutline18 />}
          title="Deploying Containers"
          {...deployingStep}
          expandable={
            deploying ? (
              <div className="bg-grayA-2">
                <DeploymentContainerLogsTable
                  logs={deploymentRuntimeLogs.data?.logs ?? []}
                  isLoading={deploymentRuntimeLogs.isLoading}
                />
              </div>
            ) : null
          }
        />
        <DeploymentStep icon={<IconEarthOutline18 />} title="Assigning Domains" {...networkStep} />
        <DeploymentStep
          icon={<IconSparkle3Outline18 />}
          title="Deployment finalizing"
          {...finalizingStep}
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

function BuildErrorDescription({
  error,
  onViewLogs,
}: {
  error: string;
  onViewLogs: () => void;
}) {
  return (
    <div className="flex items-center gap-2 min-w-0 max-w-[600px]">
      <span className="truncate min-w-0">{error}</span>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          onViewLogs();
        }}
        className="relative shrink-0 cursor-pointer underline hover:text-gray-12 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7 rounded before:absolute before:-inset-x-3 before:-inset-y-2 before:content-['']"
      >
        View full error
      </button>
    </div>
  );
}
