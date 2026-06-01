"use client";
import { LoadingState } from "@/components/loading-state";
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
import { CreateAppStep } from "./steps/create-app";
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

  // Entry points (captured once at mount so the step structure stays stable
  // even as we write projectId/appId back into the URL mid-wizard):
  //  - bare /projects/new            → step 1 creates the project (+ default app)
  //  - /projects/new?projectId=      → step 1 creates a new app in that project
  //  - /projects/new?projectId=&appId= → app exists (configuring the default app,
  //    or GitHub callback round-trip): skip step 1, start at configuration.
  const [initialProjectId] = useState(() => searchParams.get("projectId") ?? undefined);
  const [initialAppId] = useState(() => searchParams.get("appId") ?? undefined);
  // Step id to start the wizard at (e.g. "select-repo"). When the GitHub
  // callback redirects here, earlier steps are already complete so we skip ahead.
  const [initialStep] = useState(() => searchParams.get("step") ?? undefined);
  const hasInitialApp = Boolean(initialAppId);

  const [projectId, setProjectId] = useState<string | null>(initialProjectId ?? null);
  const [appId, setAppId] = useState<string | null>(initialAppId ?? null);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);

  const { bypass } = usePreventLeave(!deploymentId);

  // Persist the created project/app into the URL so a mid-wizard reload resumes
  // on the same app instead of re-running step 1 and creating a duplicate.
  // history.replaceState avoids a router re-render that would remount steps.
  const persistWizardUrl = (pid: string, aid: string | null) => {
    const params = new URLSearchParams({ projectId: pid });
    if (aid) {
      params.set("appId", aid);
    }
    window.history.replaceState(null, "", `/${workspace.slug}/projects/new?${params.toString()}`);
  };

  const handleSkipGithubSetup = () => {
    if (!projectId || !appId) {
      return;
    }
    bypass();
    router.replace(`/${workspace.slug}/projects/${projectId}/apps/${appId}/deployments`);
  };

  // Wait for creation context so the start step (which depends on whether a
  // GitHub app is already installed) is correct on first mount.
  if (contextLoading) {
    return <LoadingState message="Loading..." />;
  }

  const firstConfigStep = hasGithubInstallation ? "select-repo" : "connect-github";
  const defaultStepId = initialStep ?? (hasInitialApp ? firstConfigStep : undefined);

  return (
    <StepWizard.Root defaultStepId={defaultStepId}>
      {!hasInitialApp && (
        <StepWizard.Step id="create" label={initialProjectId ? "Create app" : "Create project"}>
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title={isFirstTimeOnboarding ? "Deploy your first app" : "Deploy your app"}
              showIconRow
              subtitle={
                <>
                  Connect a GitHub repo and get a live URL in minutes.
                  <br />
                  Unkey handles builds, infra, scaling, and routing.
                </>
              }
            />
            {initialProjectId ? (
              <CreateAppStep
                projectId={initialProjectId}
                onAppCreated={(aid) => {
                  setAppId(aid);
                  persistWizardUrl(initialProjectId, aid);
                }}
              />
            ) : (
              <CreateProjectStep
                onProjectCreated={(pid, aid) => {
                  setProjectId(pid);
                  setAppId(aid);
                  persistWizardUrl(pid, aid);
                }}
              />
            )}
          </OnboardingStepContainer>
        </StepWizard.Step>
      )}
      {!hasGithubInstallation && (
        <StepWizard.Step id="connect-github" label="Connect GitHub">
          {projectId && appId ? (
            <OnboardingStepContainer>
              <OnboardingStepHeader
                title={isFirstTimeOnboarding ? "Deploy your first app" : "Deploy your app"}
                showIconRow
                subtitle={
                  <>
                    Connect a GitHub repo and get a live URL in minutes.
                    <br />
                    Unkey handles builds, infra, scaling, and routing.
                  </>
                }
              />
              <ConnectGithubStep projectId={projectId} appId={appId} onBeforeNavigate={bypass} />
            </OnboardingStepContainer>
          ) : null}
        </StepWizard.Step>
      )}
      <StepWizard.Step id="select-repo" label="Select repository" kind="optional">
        {projectId && appId ? (
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title="Select a repository"
              subtitle={
                <>
                  Choose a repository and a branch containing your app.
                  <br />
                  We'll automatically detect Dockerfiles.
                </>
              }
            />
            <SelectRepo
              projectId={projectId}
              appId={appId}
              onBeforeNavigate={bypass}
              hasGithubInstallation={context?.hasGithubInstallation ?? false}
              onSkip={handleSkipGithubSetup}
            />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
      <StepWizard.Step id="configure-deployment" label="Configure deployment">
        {projectId && appId ? (
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title="Configure deployment"
              subtitle="Review the defaults. Edit anything you'd like to adjust."
              allowBack
            />
            <ConfigureDeploymentStep projectId={projectId} appId={appId} />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
      <StepWizard.Step id="configure-env-vars" label="Configure environment variables">
        {projectId && appId ? (
          <OnboardingStepContainer>
            <EnvVarsStep
              projectId={projectId}
              appId={appId}
              onDeploymentCreated={setDeploymentId}
            />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
      <StepWizard.Step id="deploying" label="Deploying" preventBack>
        {projectId && appId && deploymentId ? (
          <DeploymentLiveStep projectId={projectId} appId={appId} deploymentId={deploymentId} />
        ) : null}
      </StepWizard.Step>
    </StepWizard.Root>
  );
};
