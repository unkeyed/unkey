"use client";

import { ProjectDataProvider } from "../../../[projectId]/(overview)/data-provider";
import { ConfigureDeploymentContent } from "./content";
import { OnboardingEnvironmentSettingsProvider } from "./environment-provider";
import { ConfigureDeploymentFallback } from "./fallback";

type ConfigureDeploymentStepProps = {
  projectId: string;
  isFirstTimeOnboarding: boolean;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const ConfigureDeploymentStep = ({
  projectId,
  isFirstTimeOnboarding,
  onDeploymentCreated,
}: ConfigureDeploymentStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <OnboardingEnvironmentSettingsProvider>
        <ConfigureDeploymentContent
          projectId={projectId}
          isFirstTimeOnboarding={isFirstTimeOnboarding}
          onDeploymentCreated={onDeploymentCreated}
        />
      </OnboardingEnvironmentSettingsProvider>
      <ConfigureDeploymentFallback />
    </ProjectDataProvider>
  );
};
