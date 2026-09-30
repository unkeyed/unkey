"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { PRODUCTION_ENVIRONMENT_SLUG } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import {
  DeploymentLayoutProvider,
  useDeployment,
} from "../../[appId]/[environmentSlug]/deployments/[deploymentId]/layout-provider";
import { ProjectDataProvider, useProjectData } from "../../[appId]/data-provider";

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
  const { environments } = useProjectData();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();

  const deploymentUrl = routes.projects.apps.deployment({
    workspaceSlug: workspace.slug,
    projectId,
    appId,
    environmentSlug:
      environments.find((e) => e.id === deployment.environmentId)?.slug ??
      PRODUCTION_ENVIRONMENT_SLUG,
    deploymentId: deployment.id,
  });

  useEffect(() => {
    router.replace(deploymentUrl);
  }, [router, deploymentUrl]);

  return null;
};
