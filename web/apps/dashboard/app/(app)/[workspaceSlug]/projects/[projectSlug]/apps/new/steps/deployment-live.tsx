"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { ProjectDataProvider, useProjectData } from "../../[appSlug]/(overview)/data-provider";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[appSlug]/(overview)/deployments/[deploymentId]/layout-provider";

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

  const deploymentUrl = `/${workspace.slug}/projects/${projectSlug}/apps/${appSlug}/deployments/${deployment.id}`;

  useEffect(() => {
    router.replace(deploymentUrl);
  }, [router, deploymentUrl]);

  return null;
};
