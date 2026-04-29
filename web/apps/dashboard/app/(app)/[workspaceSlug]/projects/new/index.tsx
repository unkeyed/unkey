"use client";
import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { StepWizard } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { OnboardingStepContainer } from "./onboarding-step-container";
import { OnboardingStepHeader } from "./onboarding-step-header";
import { ConfigureDeploymentStep } from "./steps/configure-deployment";
import { ConnectGithubStep } from "./steps/connect-github";
import { CreateProjectStep } from "./steps/create-project";
import { DeploymentLiveStep } from "./steps/deployment-live";
import { EnvVarsStep } from "./steps/env-vars";
import { SelectRepo } from "./steps/select-repo";

export const Onboarding = () => {
  const { data: context, isLoading: contextLoading } =
    trpc.deploy.project.creationContext.useQuery();
  const isFirstTimeOnboarding = contextLoading || (context?.isFirstProject ?? true);
  const hasGithubInstallation = context?.hasGithubInstallation === true;
  const searchParams = useSearchParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  // Step id to start the wizard at (e.g. "select-repo"). When the GitHub
  // callback redirects here, earlier steps are already complete so we skip ahead.
  const initialStep = searchParams.get("step") ?? undefined;
  // Project id created in step 1. Steps 3+ need this to render; passing it
  // via URL avoids re-running the create-project step.
  const initialProjectId = searchParams.get("projectId") ?? undefined;

  const [projectId, setProjectId] = useState<string | null>(initialProjectId ?? null);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);

  const { bypass } = usePreventLeave(!deploymentId);

  const handleSkipGithubSetup = () => {
    if (!projectId) {
      return;
    }
    bypass();
    router.replace(`/${workspace.slug}/projects/${projectId}`);
  };

  return (
    <StepWizard.Root defaultStepId={initialStep}>
      <StepWizard.Step id="create-project" label="Create project">
        <OnboardingStepContainer>
          <OnboardingStepHeader
            title={isFirstTimeOnboarding ? "Deploy your first project" : "Deploy your project"}
            showIconRow
            subtitle={
              <>
                Connect a GitHub repo and get a live URL in minutes.
                <br />
                Unkey handles builds, infra, scaling, and routing.
              </>
            }
          />
          <CreateProjectStep onProjectCreated={setProjectId} />
        </OnboardingStepContainer>
      </StepWizard.Step>
      {!hasGithubInstallation && (
        <StepWizard.Step id="connect-github" label="Connect GitHub">
          {projectId ? (
            <OnboardingStepContainer>
              <OnboardingStepHeader
                title={isFirstTimeOnboarding ? "Deploy your first project" : "Deploy your project"}
                showIconRow
                subtitle={
                  <>
                    Connect a GitHub repo and get a live URL in minutes.
                    <br />
                    Unkey handles builds, infra, scaling, and routing.
                  </>
                }
              />
              <ConnectGithubStep projectId={projectId} onBeforeNavigate={bypass} />
            </OnboardingStepContainer>
          ) : null}
        </StepWizard.Step>
      )}
      <StepWizard.Step id="select-repo" label="Select repository" kind="optional">
        {projectId ? (
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title="Select a repository"
              subtitle={
                <>
                  Choose a repository and a branch containing your project.
                  <br />
                  We'll automatically detect Dockerfiles.
                </>
              }
            />
            <SelectRepo
              projectId={projectId}
              onBeforeNavigate={bypass}
              hasGithubInstallation={context?.hasGithubInstallation ?? false}
              onSkip={handleSkipGithubSetup}
            />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
      <StepWizard.Step id="configure-deployment" label="Configure deployment">
        {projectId ? (
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title="Configure deployment"
              subtitle="Review the defaults. Edit anything you'd like to adjust."
              allowBack
            />
            <ConfigureDeploymentStep projectId={projectId} />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
      <StepWizard.Step id="configure-env-vars" label="Configure environment variables">
        {projectId ? (
          <OnboardingStepContainer>
            <EnvVarsStep projectId={projectId} onDeploymentCreated={setDeploymentId} />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
      <StepWizard.Step id="deploying" label="Deploying" preventBack>
        {projectId && deploymentId ? (
          <DeploymentLiveStep projectId={projectId} deploymentId={deploymentId} />
        ) : null}
      </StepWizard.Step>
    </StepWizard.Root>
  );
};
