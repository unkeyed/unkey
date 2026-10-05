"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { collection } from "@/lib/collections";
import { dismissSettingsBanner } from "@/lib/collections/deploy/environment-settings";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { SourceKind } from "../../wizard-model";
import { firstDeployEnvironment } from "./deploy-target";

export function useFirstDeploy(
  projectId: string,
  appId: string,
  source: SourceKind,
  onDeploymentCreated: (deploymentId: string) => void,
) {
  const { environments, refetchDeployments } = useProjectData();
  const environment = firstDeployEnvironment(source, environments);

  const deploy = useMutation({
    mutationFn: async (target: string) => {
      const res = await getUnkeyClient().deployments.createDeploymentV3({
        project: projectId,
        app: appId,
        environment: target,
      });
      return res.data.deploymentId;
    },
    onSuccess: (deploymentId) => {
      onDeploymentCreated(deploymentId);
      dismissSettingsBanner();
      refetchDeployments();
      collection.apps.utils.refetch().catch(() => undefined);
    },
    onError: (error) => {
      toast.error("Could not start the deployment", { description: getErrorMessage(error) });
    },
  });

  return {
    canDeploy: environment !== null && !deploy.isLoading,
    isDeploying: deploy.isLoading,
    start: () => {
      if (environment) {
        deploy.mutate(environment);
      }
    },
  };
}
