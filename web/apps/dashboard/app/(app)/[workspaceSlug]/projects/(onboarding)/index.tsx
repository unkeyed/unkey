"use client";
import { StepWizard } from "@unkey/ui";
import { useState } from "react";
import { OnboardingHeader } from "./onboarding-header";
import { ConnectGithubStep } from "./steps/connect-github";
import { CreateProjectStep } from "./steps/create-project";

export const Onboarding = () => {
  const [projectId, setProjectId] = useState<string | null>(null);

  return (
    <div className="flex flex-col items-center justify-center h-screen relative">
      <StepWizard.Root>
        <OnboardingHeader projectId={projectId} />
        <StepWizard.Step id="create-project" label="Create project">
          <CreateProjectStep onProjectCreated={setProjectId} />
        </StepWizard.Step>
        <StepWizard.Step id="connect-github" label="Connect GitHub">
          <ConnectGithubStep projectId={projectId} />
        </StepWizard.Step>
      </StepWizard.Root>
    </div>
  );
};
