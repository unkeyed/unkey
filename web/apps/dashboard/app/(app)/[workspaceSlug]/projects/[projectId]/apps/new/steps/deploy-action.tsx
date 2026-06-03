"use client";

import { queryClient } from "@/lib/collections/client";
import { trpc } from "@/lib/trpc/client";
import { Button, toast, useStepWizard } from "@unkey/ui";

type DeployActionProps = {
  projectId: string;
  appId: string;
  disabled?: boolean;
  onDeploymentCreated: (deploymentId: string) => void;
};

export const DeployAction = ({
  projectId,
  appId,
  disabled,
  onDeploymentCreated,
}: DeployActionProps) => {
  const { goTo } = useStepWizard();

  const deploy = trpc.deploy.deployment.create.useMutation({
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      toast.success("Deployment triggered", {
        description: "Your app is being built and deployed",
      });
      onDeploymentCreated(data.deploymentId);
      goTo("deploying");
    },
    onError: (error) => {
      toast.error("Deployment failed", { description: error.message });
    },
  });

  return (
    <div className="flex justify-end mt-6 flex-col gap-4">
      <Button
        type="button"
        variant="primary"
        size="xlg"
        className="rounded-lg"
        disabled={deploy.isLoading || disabled}
        loading={deploy.isLoading}
        onClick={() => deploy.mutate({ projectId, appId, environmentSlug: "production" })}
      >
        Deploy
      </Button>
      <span className="text-gray-10 text-[13px] text-center">
        We'll build your image, provision infrastructure, and more.
      </span>
    </div>
  );
};
