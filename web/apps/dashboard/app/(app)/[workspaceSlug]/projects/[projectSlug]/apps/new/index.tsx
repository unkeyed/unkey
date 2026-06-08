"use client";
import { LoadingState } from "@/components/loading-state";
import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { useProjectSlug } from "@/hooks/use-route-slugs";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { StepWizard } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { OnboardingStepContainer } from "./onboarding-step-container";
import { OnboardingStepHeader } from "./onboarding-step-header";
import { ConfigureDeploymentStep } from "./steps/configure-deployment";
import { ConnectGithubStep } from "./steps/connect-github";
import { CreateAppStep } from "./steps/create-app";
import { DeploymentLiveStep } from "./steps/deployment-live";
import { EnvVarsStep } from "./steps/env-vars";
import { SelectRepo } from "./steps/select-repo";

export const AppSetupWizard = () => {
  const { data: context } = trpc.deploy.project.creationContext.useQuery();
  const hasGithubInstallation = context?.hasGithubInstallation === true;
  const searchParams = useSearchParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const projectSlug = useProjectSlug();
  const projectQuery = useLiveQuery(
    (q) =>
      projectSlug
        ? q
            .from({ project: collection.projects })
            .where(({ project }) => eq(project.slug, projectSlug))
        : undefined,
    [projectSlug],
  );
  const projectId = projectQuery.data?.at(0)?.id;

  // Step id to start the wizard at (e.g. "select-repo"). When the GitHub
  // callback redirects here, earlier steps are already complete so we skip ahead.
  const initialStep = searchParams.get("step") ?? undefined;
  // App id created in the first step. Later steps need it; the GitHub install
  // round-trip carries it back via the signed state and ?appId= so it survives
  // the full-page redirect.
  const initialAppId = searchParams.get("appId") ?? undefined;

  const [appId, setAppId] = useState<string | null>(initialAppId ?? null);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);

  const { bypass } = usePreventLeave(!deploymentId);

  const handleSkipGithubSetup = () => {
    bypass();
    router.replace(`/${workspace.slug}/projects/${projectSlug}`);
  };

  if (!projectSlug) {
    console.warn("[AppSetupWizard] missing projectSlug route param");
    return <LoadingState />;
  }
  // Steps thread projectId into mutations immediately; wait for the slug to
  // resolve before mounting the wizard.
  if (!projectId) {
    return <LoadingState />;
  }

  return (
    <StepWizard.Root defaultStepId={initialStep}>
      <StepWizard.Step id="create-app" label="Create app">
        <OnboardingStepContainer>
          {deployYourAppHeader}
          <CreateAppStep projectId={projectId} projectSlug={projectSlug} onAppCreated={setAppId} />
        </OnboardingStepContainer>
      </StepWizard.Step>
      {!hasGithubInstallation && (
        <StepWizard.Step id="connect-github" label="Connect GitHub">
          {appId ? (
            <OnboardingStepContainer>
              {deployYourAppHeader}
              <ConnectGithubStep projectId={projectId} appId={appId} onBeforeNavigate={bypass} />
            </OnboardingStepContainer>
          ) : null}
        </StepWizard.Step>
      )}
      <StepWizard.Step id="select-repo" label="Select repository" kind="optional">
        {appId ? (
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
        {appId ? (
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
        {appId ? (
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
        {appId && deploymentId ? (
          <DeploymentLiveStep projectId={projectId} appId={appId} deploymentId={deploymentId} />
        ) : null}
      </StepWizard.Step>
    </StepWizard.Root>
  );
};

const deployYourAppHeader = (
  <OnboardingStepHeader
    title="Deploy your app"
    showIconRow
    subtitle={
      <>
        Connect a GitHub repo and get a live URL in minutes.
        <br />
        Unkey handles builds, infra, scaling, and routing.
      </>
    }
  />
);
