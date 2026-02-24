"use client";
import { StepWizard } from "@unkey/ui";
import { OnboardingHeader } from "./onboarding-header";
import { ConnectGithubStep } from "./steps/connect-github";
import { CreateProjectStep } from "./steps/create-project";

export const Onboarding = () => (
  <div className="flex flex-col items-center justify-center h-screen relative">
    <StepWizard.Root>
      <OnboardingHeader />
      <StepWizard.Step id="create-project" label="Create project">
        <CreateProjectStep />
      </StepWizard.Step>
      <StepWizard.Step id="connect-github" label="Connect GitHub">
        <ConnectGithubStep />
      </StepWizard.Step>
    </StepWizard.Root>
  </div>
);
