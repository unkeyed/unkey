"use client";

import { trpc } from "@/lib/trpc/client";
import { Check } from "@unkey/icons";
import { useMemo } from "react";
import { ProjectDataProvider } from "../../[projectId]/(overview)/data-provider";
import { DeploymentInfo } from "../../[projectId]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-info";
import { DeploymentProgress } from "../../[projectId]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-progress";
import { deriveStatusFromSteps } from "../../[projectId]/(overview)/deployments/[deploymentId]/deployment-utils";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[projectId]/(overview)/deployments/[deploymentId]/layout-provider";
import { OnboardingStepContainer } from "../onboarding-step-container";
import { OnboardingStepHeader } from "../onboarding-step-header";

type DeploymentLiveStepProps = {
  projectId: string;
  deploymentId: string;
};

export const DeploymentLiveStep = ({ projectId, deploymentId }: DeploymentLiveStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <DeploymentLayoutProvider deploymentId={deploymentId}>
        <DeploymentLiveStepContent />
      </DeploymentLayoutProvider>
    </ProjectDataProvider>
  );
};

const DeploymentLiveStepContent = () => {
  const { deployment } = useDeployment();
  const ready = deployment.status === "ready";

  const stepsQuery = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    { refetchInterval: ready ? false : 1_000, refetchOnWindowFocus: false },
  );

  const derivedStatus = useMemo(
    () => deriveStatusFromSteps(stepsQuery.data, deployment.status),
    [stepsQuery.data, deployment.status],
  );

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
