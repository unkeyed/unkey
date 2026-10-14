"use client";

import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { EnvironmentSettingsProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/environment-provider";
import { ConfigureDeploymentContent } from "./content";
import { ConfigureDeploymentFallback } from "./fallback";

type ConfigureDeploymentStepProps = {
  projectId: string;
  appId: string;
  onDeploymentCreated: (deploymentId: string) => void;
};

export function ConfigureDeploymentStep({
  projectId,
  appId,
  onDeploymentCreated,
}: ConfigureDeploymentStepProps) {
  return (
    <ProjectDataProvider projectId={projectId} appId={appId}>
      <EnvironmentSettingsProvider
        fallback={<ConfigureDeploymentFallback onDeploymentCreated={onDeploymentCreated} />}
      >
        <ConfigureDeploymentContent onDeploymentCreated={onDeploymentCreated} />
      </EnvironmentSettingsProvider>
    </ProjectDataProvider>
  );
}
