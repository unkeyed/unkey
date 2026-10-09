"use client";

import {
  useAppId,
  useProjectData,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, toast, useStepWizard } from "@unkey/ui";

type DeployState = "loading" | "dirty" | "ready";

const DEPLOY_STATE: Record<DeployState, { disabled: boolean; note: string }> = {
  loading: { disabled: true, note: "Loading your deployment settings" },
  dirty: { disabled: true, note: "Save your changes to deploy" },
  ready: { disabled: false, note: "We'll build your image, provision infrastructure, and more." },
};

type DeployActionProps = {
  state: DeployState;
  onDeploymentCreated: (deploymentId: string) => void;
};

export function DeployAction({ state, onDeploymentCreated }: DeployActionProps) {
  const { disabled, note } = DEPLOY_STATE[state];
  const { goTo } = useStepWizard();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const { projectId, environments, refetchDeployments } = useProjectData();
  const appId = useAppId();
  const productionEnvironment = environments.find(
    (environment) => environment.kind === ENVIRONMENT_KIND.production,
  );

  const deploy = useMutation({
    mutationFn: async (environment: string) => {
      const res = await getUnkeyClient().deployments.createDeploymentV3({
        project: projectId,
        app: appId,
        environment,
      });
      return { deploymentId: res.data.deploymentId };
    },
    onSuccess: async (data) => {
      refetchDeployments();
      await collection.apps.utils.refetch();
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
      <span className="text-gray-10 text-sm text-center">{note}</span>
      {planGate}
    </div>
  );
}
