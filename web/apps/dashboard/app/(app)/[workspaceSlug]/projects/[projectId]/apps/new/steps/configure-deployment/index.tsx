"use client";

import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useState } from "react";
import { ConfigureDeploymentContent } from "./content";
import { OnboardingEnvironmentSettingsProvider } from "./environment-provider";
import { ConfigureDeploymentFallback } from "./fallback";

type ConfigureDeploymentStepProps = {
  projectId: string;
  appId: string;
};

export const ConfigureDeploymentStep = ({ projectId, appId }: ConfigureDeploymentStepProps) => {
  const [settingsReady, setSettingsReady] = useState(false);

  return (
    <ProjectDataProvider projectId={projectId} appId={appId}>
      <OnboardingEnvironmentSettingsProvider onSettingsReady={() => setSettingsReady(true)}>
        <ConfigureDeploymentContent />
      </OnboardingEnvironmentSettingsProvider>
      <ConfigureDeploymentFallback settingsReady={settingsReady} />
    </ProjectDataProvider>
  );
};
