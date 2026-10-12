"use client";
import { LeaveGuard, useLeaveBypass } from "@/components/leave-guard";
import { OnboardingStepContainer } from "@/components/onboarding/step-container";
import { OnboardingStepHeader } from "@/components/onboarding/step-header";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  StepWizard,
  UnsavedChangesScope,
  useReportUnsavedChanges,
  useUnsavedChanges,
} from "@unkey/ui";
import type { Route } from "next";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { ChooseSourceStep } from "./steps/choose-source";
import { ConfigureDeploymentStep } from "./steps/configure-deployment";
import { type AppDetails, CreateAppStep } from "./steps/create-app";
import { DeploymentLiveStep } from "./steps/deployment-live";
import { SelectRepo } from "./steps/select-repo";

export default function AppSetupPage() {
  const params = useParams();
  const workspace = useWorkspaceNavigation();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const projectHref = routes.projects.detail({ workspaceSlug: workspace.slug, projectId });

  return (
    <LeaveGuard fallbackHref={projectHref}>
      <AppSetup projectId={projectId} projectHref={projectHref} />
    </LeaveGuard>
  );
}

function AppSetup({ projectId, projectHref }: { projectId: string; projectHref: Route }) {
  const { data: context } = trpc.deploy.project.creationContext.useQuery();
  const searchParams = useSearchParams();
  const router = useRouter();

  // Step id to start the wizard at (e.g. "select-repo"). When the GitHub
  // callback redirects here, earlier steps are already complete so we skip ahead.
  const initialStep = searchParams.get("step") ?? undefined;
  // App id created in the first step. Later steps need it; the GitHub install
  // round-trip carries it back via the signed state and ?appId= so it survives
  // the full-page redirect.
  const initialAppId = searchParams.get("appId") ?? undefined;

  const [appId, setAppId] = useState<string | null>(initialAppId ?? null);
  const [appDetails, setAppDetails] = useState<AppDetails | null>(null);
  const [deploymentId, setDeploymentId] = useState<string | null>(null);

  const { isDirty, report } = useUnsavedChanges();
  useReportUnsavedChanges((isDirty || appId !== null) && !deploymentId);
  const bypass = useLeaveBypass();

  const handleSkipGithubSetup = () => {
    bypass();
    router.replace(projectHref);
  };

  return (
    <UnsavedChangesScope report={report}>
      <StepWizard.Root defaultStepId={initialStep}>
        <StepWizard.Step id="create-app" label="Create app">
          <OnboardingStepContainer>
            {deployYourAppHeader}
            <CreateAppStep onAppDetailsSubmitted={setAppDetails} />
          </OnboardingStepContainer>
        </StepWizard.Step>
        <StepWizard.Step id="choose-source" label="Choose source">
          {appDetails ? (
            <OnboardingStepContainer>
              {deployYourAppHeader}
              <ChooseSourceStep
                projectId={projectId}
                appDetails={appDetails}
                onAppCreated={setAppId}
                onBeforeNavigate={bypass}
              />
            </OnboardingStepContainer>
          ) : null}
        </StepWizard.Step>
        <StepWizard.Step id="select-repo" label="Select repository" kind="optional">
          {appId ? (
            <OnboardingStepContainer>
              <OnboardingStepHeader
                title="Select a repository"
                subtitle={
                  <>
                    Choose a repository and a branch containing your app.
                    <br />
                    We'll detect how to build it automatically.
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
            <ConfigureDeploymentStep
              projectId={projectId}
              appId={appId}
              onDeploymentCreated={(id) => {
                bypass();
                setDeploymentId(id);
              }}
            />
          ) : null}
        </StepWizard.Step>
        <StepWizard.Step id="deploying" label="Deploying" preventBack>
          {appId && deploymentId ? (
            <DeploymentLiveStep projectId={projectId} appId={appId} deploymentId={deploymentId} />
          ) : null}
        </StepWizard.Step>
      </StepWizard.Root>
    </UnsavedChangesScope>
  );
}

const deployYourAppHeader = (
  <OnboardingStepHeader
    title="Deploy your app"
    showIconRow
    subtitle={
      <>
        Connect a GitHub repo or deploy a published image, and get a live URL in minutes.
        <br />
        Unkey handles builds, infra, scaling, and routing.
      </>
    }
  />
);
