"use client";

import {
  useAppId,
  useProjectData,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND, type Environment } from "@/lib/collections/deploy/environments";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { Button, toast, useStepWizard } from "@unkey/ui";

type DeployState =
  | { kind: "loading" }
  | { kind: "no-environment" }
  | { kind: "dirty" }
  | { kind: "ready"; environmentSlug: string };

const DEPLOY_STATE: Record<DeployState["kind"], { disabled: boolean; note: string }> = {
  loading: { disabled: true, note: "Loading your deployment settings" },
  "no-environment": { disabled: true, note: "This app has no production environment to deploy to" },
  dirty: { disabled: true, note: "Save your changes to deploy" },
  ready: { disabled: false, note: "We'll build your image, provision infrastructure, and more." },
};

function resolveDeployState({
  loading,
  dirty,
  environments,
}: {
  loading: boolean;
  dirty: boolean;
  environments: Environment[];
}): DeployState {
  if (loading) {
    return { kind: "loading" };
  }
  const production = environments.find(
    (environment) => environment.kind === ENVIRONMENT_KIND.production,
  );
  if (!production) {
    return { kind: "no-environment" };
  }
  return dirty ? { kind: "dirty" } : { kind: "ready", environmentSlug: production.slug };
}

type DeployActionProps = {
  loading?: boolean;
  dirty?: boolean;
  onDeploymentCreated: (deploymentId: string) => void;
};

export function DeployAction({
  loading = false,
  dirty = false,
  onDeploymentCreated,
}: DeployActionProps) {
  const { goTo } = useStepWizard();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const { projectId, environments, refetchDeployments } = useProjectData();
  const appId = useAppId();
  const state = resolveDeployState({ loading, dirty, environments });
  const { disabled, note } = DEPLOY_STATE[state.kind];

  const deploy = useMutation({
    mutationFn: async (environment: string) => {
      const res = await getUnkeyClient().deployments.createDeploymentV3({
        project: projectId,
        app: appId,
        environment,
      });
      return { deploymentId: res.data.deploymentId };
    },
    onSuccess: (data) => {
      refetchDeployments();
      collection.apps.utils.refetch().catch(() => undefined);
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
        disabled={deploy.isLoading || disabled}
        loading={deploy.isLoading}
        onClick={() => {
          if (gated) {
            openPaywall();
          } else if (state.kind === "ready") {
            deploy.mutate(state.environmentSlug);
          }
        }}
      >
        Deploy
      </Button>
      <span className="text-gray-10 text-sm text-center">{note}</span>
      {planGate}
    </div>
  );
}
