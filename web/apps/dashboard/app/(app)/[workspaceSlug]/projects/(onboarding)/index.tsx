"use client";
import { StepWizard } from "@unkey/ui";
import { useState } from "react";
import { OnboardingHeader } from "./onboarding-header";
import { SelectRepo } from "./steps/select-repo";

export const Onboarding = () => {
  const [projectId, _] = useState<string | null>(null);

  return (
    <div className="flex flex-col items-center justify-center h-screen relative">
      <StepWizard.Root>
        <OnboardingHeader projectId={projectId} />
        {/* <StepWizard.Step id="create-project" label="Create project"> */}
        {/*   <CreateProjectStep onProjectCreated={setProjectId} /> */}
        {/* </StepWizard.Step> */}
        {/* <StepWizard.Step id="connect-github" label="Connect GitHub"> */}
        {/*   <ConnectGithubStep projectId={projectId} /> */}
        {/* </StepWizard.Step> */}
        <StepWizard.Step id="select-repo" label="Connect GitHub">
          {/* // Clean this up later. ProjectId cannot be null after the first step  */}
          <SelectRepo projectId={projectId ?? undefined} />
        </StepWizard.Step>
      </StepWizard.Root>
    </div>
  );
};
