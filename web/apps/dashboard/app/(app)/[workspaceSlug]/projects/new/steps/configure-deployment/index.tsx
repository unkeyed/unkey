"use client";

import { ProjectDataProvider } from "../../../[projectId]/(overview)/data-provider";
import { OnboardingEnvironmentSettingsProvider } from "./environment-provider";
import { ConfigureDeploymentContent } from "./content";
import { ConfigureDeploymentFallback } from "./fallback";

type ConfigureDeploymentStepProps = {
  projectId: string;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const ConfigureDeploymentStep = ({
  projectId,
  onDeploymentCreated,
}: ConfigureDeploymentStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <OnboardingEnvironmentSettingsProvider>
        <ConfigureDeploymentContent
          projectId={projectId}
          onDeploymentCreated={onDeploymentCreated}
        />
      </OnboardingEnvironmentSettingsProvider>
      <ConfigureDeploymentFallback />
    </ProjectDataProvider>
  );
};
