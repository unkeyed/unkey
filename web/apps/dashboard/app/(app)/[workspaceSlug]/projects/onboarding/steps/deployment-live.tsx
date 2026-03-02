"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Check } from "@unkey/icons";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ProjectDataProvider } from "../../[projectId]/(overview)/data-provider";
import { DeploymentInfo } from "../../[projectId]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-info";
import { DeploymentProgress } from "../../[projectId]/(overview)/deployments/[deploymentId]/(deployment-progress)/deployment-progress";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[projectId]/(overview)/deployments/[deploymentId]/layout-provider";
import { OnboardingStepHeader } from "../onboarding-step-header";

type DeploymentLiveStepProps = {
  projectId: string;
  deploymentId: string;
};

export const DeploymentLiveStep = ({ projectId, deploymentId }: DeploymentLiveStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <DeploymentLayoutProvider deploymentId={deploymentId}>
        <DeploymentLiveStepContent projectId={projectId} />
      </DeploymentLayoutProvider>
    </ProjectDataProvider>
  );
};

const REDIRECT_DELAY_SECONDS = 15;

const DeploymentLiveStepContent = ({ projectId }: { projectId: string }) => {
  const { deployment } = useDeployment();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const ready = deployment.status === "ready";
  const [countdown, setCountdown] = useState(REDIRECT_DELAY_SECONDS);

  const deploymentUrl = `/${workspace.slug}/projects/${projectId}/deployments/${deployment.id}`;

  useEffect(() => {
    if (!ready) {
      return;
    }

    const interval = setInterval(() => {
      setCountdown((prev) => (prev <= 1 ? 0 : prev - 1));
    }, 1000);

    return () => clearInterval(interval);
  }, [ready]);

  useEffect(() => {
    if (countdown === 0) {
      router.push(deploymentUrl);
    }
  }, [countdown, router, deploymentUrl]);

  return (
    <div className="flex flex-col items-center justify-center mt-14">
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
        subtitle={
          ready ? (
            <>
              Redirecting to project overview in{" "}
              <span className="font-medium text-gray-12 inline-block w-[2ch] text-right tabular-nums">
                {countdown}
              </span>{" "}              seconds
              …{" "}
              <Link
                href={deploymentUrl}
                className="font-medium text-gray-12 border-b border-dotted"
              >
                Go now
              </Link>
            </>
          ) : (
            "Building, provisioning infrastructure, and assigning domains..."
          )
        }
      />
      <div className="w-[900px] space-y-6">
        <DeploymentInfo />
        <DeploymentProgress />
      </div>
    </div>
  );
};
