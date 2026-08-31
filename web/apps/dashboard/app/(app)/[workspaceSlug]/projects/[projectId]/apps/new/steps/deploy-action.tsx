"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { queryClient } from "@/lib/collections/client";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, toast, useStepWizard } from "@unkey/ui";
import { useProjectData } from "../../[appId]/(overview)/data-provider";

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
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const { environments } = useProjectData();
  const productionEnvironment = environments.find(
    (environment) => environment.kind === ENVIRONMENT_KIND.production,
  );

  const deploy = useMutation({
    mutationFn: async (environment: string) => {
      const res = await getUnkeyClient().deployments.createDeployment({
        idempotencyKey: crypto.randomUUID(),
        v2DeploymentsCreateDeploymentRequestBody: {
          project: projectId,
          app: appId,
          environment,
          // No branch or commitSha: the API builds the app's default branch.
          git: {},
        },
      });
      return { deploymentId: res.result.data.deploymentId };
    },
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      toast.success("Deployment triggered", {
        description: "Your app is being built and deployed",
      });
      onDeploymentCreated(data.deploymentId);
      goTo("deploying");
    },
    onError: (error) => {
      toast.error("Deployment failed", { description: getErrorMessage(error) });
    },
  });

  return (
    <div className="flex justify-end mt-6 flex-col gap-4">
      <Button
        type="button"
        variant="primary"
        size="xlg"
        className="rounded-lg"
        disabled={deploy.isLoading || disabled || !productionEnvironment}
        loading={deploy.isLoading}
        onClick={() =>
          gated ? openPaywall() : productionEnvironment && deploy.mutate(productionEnvironment.slug)
        }
      >
        Deploy
      </Button>
      <span className="text-gray-10 text-[13px] text-center">
        We'll build your image, provision infrastructure, and more.
      </span>
      {planGate}
    </div>
  );
};
