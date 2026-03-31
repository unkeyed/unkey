"use client";

import { Button, useStepWizard } from "@unkey/ui";
import { DeploymentSettings } from "../../../[projectId]/(overview)/settings/deployment-settings";
import { useEnvironmentSettings } from "../../../[projectId]/(overview)/settings/environment-provider";
import { OnboardingStepHint, OnboardingStepHintHighlight } from "../../onboarding-step-hint";
import { DeployAction } from "../deploy-action";

type ConfigureDeploymentContentProps = {
  projectId: string;
  isFirstTimeOnboarding: boolean;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const ConfigureDeploymentContent = ({
  projectId,
  isFirstTimeOnboarding,
  onDeploymentCreated,
}: ConfigureDeploymentContentProps) => {
  const { next } = useStepWizard();
  const { isSaving } = useEnvironmentSettings();

  return (
    <div className="w-225">
      <DeploymentSettings githubReadOnly sections={{ build: true }} />
      {isFirstTimeOnboarding ? (
        <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
          <Button type="button" variant="primary" size="xlg" className="rounded-lg" onClick={next}>
            Next
          </Button>
          <span className="text-gray-10 text-[13px] text-center">
            Start configuring your environment variables
            <br />
          </span>
        </div>
      ) : (
        <>
          <DeployAction
            projectId={projectId}
            disabled={isSaving}
            onDeploymentCreated={onDeploymentCreated}
          />
          <button type="button" className="cursor-pointer w-full group self-center" onClick={next}>
            <OnboardingStepHint>
              Want to set{" "}
              <OnboardingStepHintHighlight>environment variables</OnboardingStepHintHighlight>{" "}
              first?
            </OnboardingStepHint>
          </button>
        </>
      )}
    </div>
  );
};
