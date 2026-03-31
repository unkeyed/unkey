"use client";

import { ProjectDataProvider } from "../../../[projectId]/(overview)/data-provider";
import { ConfigureDeploymentContent } from "./content";
import { OnboardingEnvironmentSettingsProvider } from "./environment-provider";
import { ConfigureDeploymentFallback } from "./fallback";

type ConfigureDeploymentStepProps = {
  projectId: string;
};

export const ConfigureDeploymentStep = ({ projectId }: ConfigureDeploymentStepProps) => {
  return (
    <ProjectDataProvider projectId={projectId}>
      <OnboardingEnvironmentSettingsProvider>
        <ConfigureDeploymentContent />
      </OnboardingEnvironmentSettingsProvider>
      <ConfigureDeploymentFallback />
    </ProjectDataProvider>
  );
};
