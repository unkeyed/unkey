"use client";

import { queryClient } from "@/lib/collections/client";
import { trpc } from "@/lib/trpc/client";
import { Button, toast, useStepWizard } from "@unkey/ui";
import { DeploymentSettings } from "../../../[projectId]/(overview)/settings/deployment-settings";
import { useEnvironmentSettings } from "../../../[projectId]/(overview)/settings/environment-provider";

type ConfigureDeploymentContentProps = {
  projectId: string;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const ConfigureDeploymentContent = ({
  projectId,
  onDeploymentCreated,
}: ConfigureDeploymentContentProps) => {
  const { next } = useStepWizard();
  const { isSaving } = useEnvironmentSettings();

  const deploy = trpc.deploy.deployment.create.useMutation({
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      toast.success("Deployment triggered", {
        description: "Your project is being built and deployed",
      });
      onDeploymentCreated(data.deploymentId);
      next();
    },
    onError: (error) => {
      toast.error("Deployment failed", { description: error.message });
    },
  });

  return (
    <div className="w-225">
      <DeploymentSettings githubReadOnly sections={{ build: true }} />
      <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
        <Button
          type="button"
          variant="primary"
          size="xlg"
          className="rounded-lg"
          disabled={deploy.isLoading || isSaving}
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
  );
};
