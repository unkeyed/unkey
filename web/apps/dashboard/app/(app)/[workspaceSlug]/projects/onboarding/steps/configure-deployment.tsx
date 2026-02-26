"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Button, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { ProjectDataProvider } from "../../[projectId]/(overview)/data-provider";
import { DeploymentSettings } from "../../[projectId]/(overview)/settings/deployment-settings";
import { OnboardingEnvironmentSettingsProvider } from "./onboarding-environment-provider";

type ConfigureDeploymentStepProps = {
  projectId: string;
};

export const ConfigureDeploymentStep = ({ projectId }: ConfigureDeploymentStepProps) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const deploy = trpc.deploy.deployment.create.useMutation({
    onSuccess: (data) => {
      toast.success("Deployment triggered", {
        description: "Your project is being built and deployed",
      });
      router.push(
        `/${workspace.slug}/projects/${projectId}/deployments/${data.deploymentId}`,
      );
    },
    onError: (error) => {
      toast.error("Deployment failed", {
        description: error.message,
      });
    },
  });

  return (
    <ProjectDataProvider projectId={projectId}>
      <OnboardingEnvironmentSettingsProvider>
        <div className="w-[900px]">
          <DeploymentSettings githubReadOnly />
          <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
            <Button
              type="button"
              variant="primary"
              size="xlg"
              className="rounded-lg"
              disabled={deploy.isLoading}
              loading={deploy.isLoading}
              onClick={() => deploy.mutate({ projectId, environmentSlug: "production" })}
            >
              Deploy
            </Button>
            <span className="text-gray-10 text-[13px] text-center">
              We'll build your image, provision infrastructure, and more.
              <br />
            </span>
          </div>
        </div>
      </OnboardingEnvironmentSettingsProvider>
    </ProjectDataProvider>
  );
};
