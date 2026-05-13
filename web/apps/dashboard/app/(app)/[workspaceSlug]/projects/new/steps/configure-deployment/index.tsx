"use client";

import { useState } from "react";
import { ProjectDataProvider } from "../../../[projectId]/(overview)/data-provider";
import { ConfigureDeploymentContent } from "./content";
import { OnboardingEnvironmentSettingsProvider } from "./environment-provider";
import { ConfigureDeploymentFallback } from "./fallback";

type ConfigureDeploymentStepProps = {
  projectId: string;
};

export const ConfigureDeploymentStep = ({ projectId }: ConfigureDeploymentStepProps) => {
  const [settingsReady, setSettingsReady] = useState(false);

  return (
    <ProjectDataProvider projectId={projectId}>
      <OnboardingEnvironmentSettingsProvider onSettingsReady={() => setSettingsReady(true)}>
        <ConfigureDeploymentContent />
      </OnboardingEnvironmentSettingsProvider>
      <ConfigureDeploymentFallback settingsReady={settingsReady} />
    </ProjectDataProvider>
  );
};
