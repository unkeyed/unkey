"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Check } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useEffect, useMemo } from "react";
import { ProjectDataProvider } from "../../[appId]/(overview)/data-provider";
import { DeploymentInfo } from "../../[appId]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-info";
import { DeploymentProgress } from "../../[appId]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-progress";
import { deriveStatusFromSteps } from "../../[appId]/(overview)/deployments/[deploymentId]/deployment-utils";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[appId]/(overview)/deployments/[deploymentId]/layout-provider";
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
        <DeploymentLiveStepContent projectId={projectId} appId={appId} />
      </DeploymentLayoutProvider>
    </ProjectDataProvider>
  );
};

const DeploymentLiveStepContent = ({ projectId, appId }: { projectId: string; appId: string }) => {
  const { deployment } = useDeployment();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const ready = deployment.status === "ready";

  const deploymentUrl = `/${workspace.slug}/projects/${projectId}/apps/${appId}/deployments/${deployment.id}`;

  useEffect(() => {
    router.replace(deploymentUrl);
  }, [ready, router, deploymentUrl]);

  return null
};
