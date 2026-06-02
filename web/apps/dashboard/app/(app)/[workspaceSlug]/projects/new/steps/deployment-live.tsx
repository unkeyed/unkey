"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Check } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useEffect, useMemo } from "react";
import {
  ProjectDataProvider,
  useProjectData,
} from "../../[projectSlug]/apps/[appSlug]/(overview)/data-provider";
import { DeploymentInfo } from "../../[projectSlug]/apps/[appSlug]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-info";
import { DeploymentProgress } from "../../[projectSlug]/apps/[appSlug]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-progress";
import { deriveStatusFromSteps } from "../../[projectSlug]/apps/[appSlug]/(overview)/deployments/[deploymentId]/deployment-utils";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[projectSlug]/apps/[appSlug]/(overview)/deployments/[deploymentId]/layout-provider";
import { OnboardingStepContainer } from "../onboarding-step-container";
import { OnboardingStepHeader } from "../onboarding-step-header";

type DeploymentLiveStepProps = {
  projectId: string;
  appId: string;
  deploymentId: string;
};

export const DeploymentLiveStep = ({ projectId, appId, deploymentId }: DeploymentLiveStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId} appId={appId}>
      <DeploymentLayoutProvider deploymentId={deploymentId}>
        <DeploymentLiveStepContent />
      </DeploymentLayoutProvider>
    </ProjectDataProvider>
  );
};

const DeploymentLiveStepContent = () => {
  const { deployment } = useDeployment();
  const { projectSlug, appSlug } = useProjectData();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const ready = deployment.status === "ready";

  const stepsQuery = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    { refetchInterval: ready ? false : 1_000, refetchOnWindowFocus: false },
  );

  const derivedStatus = useMemo(
    () => deriveStatusFromSteps(stepsQuery.data, deployment.status),
    [stepsQuery.data, deployment.status],
  );

  const canRedirect = Boolean(projectSlug) && Boolean(appSlug);
  const deploymentUrl = `/${workspace.slug}/projects/${projectSlug}/apps/${appSlug}/deployments/${deployment.id}`;

  useEffect(() => {
    if (ready && canRedirect) {
      router.replace(deploymentUrl);
    }
  }, [ready, canRedirect, router, deploymentUrl]);

  return (
    <OnboardingStepContainer>
      <OnboardingStepHeader
        title={
          ready ? (
            <span className="flex items-center gap-3">
              Deployment complete!
              <Check iconSize="md-regular" className="text-success-11" />
            </span>
          ) : (
            "Deploying your project"
          )
        }
        subtitle="Building, provisioning infrastructure, and assigning domains..."
      />
      <div className="w-[900px] space-y-6">
        <DeploymentInfo statusOverride={derivedStatus} />
        <DeploymentProgress stepsData={stepsQuery.data} />
      </div>
    </OnboardingStepContainer>
  );
};
