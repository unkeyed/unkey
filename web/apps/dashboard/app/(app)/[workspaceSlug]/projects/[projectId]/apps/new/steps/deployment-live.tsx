"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { deploymentPath } from "@/lib/navigation/routes/projects";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { ProjectDataProvider } from "../../[appId]/(overview)/data-provider";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[appId]/(overview)/deployments/[deploymentId]/layout-provider";

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

  const deploymentUrl = deploymentPath({
    workspaceSlug: workspace.slug,
    projectId,
    appId,
    deploymentId: deployment.id,
  });

  useEffect(() => {
    router.replace(deploymentUrl);
  }, [router, deploymentUrl]);

  return null;
};
