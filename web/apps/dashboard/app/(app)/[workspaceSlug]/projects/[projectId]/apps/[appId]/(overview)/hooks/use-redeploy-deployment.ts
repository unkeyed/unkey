"use client";

import { useGuardedNavigate } from "@/components/leave-guard";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { dismissSettingsBanner } from "@/lib/collections/deploy/pending-redeploy";
import { routes } from "@/lib/navigation/routes";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useProjectData } from "../data-provider";

export type RedeploySource = {
  id: string;
  projectId: string;
  appId: string;
  environmentId: string;
};

export function useRedeployDeployment() {
  const { refetchDeployments } = useProjectData();
  const navigate = useGuardedNavigate();
  const workspace = useWorkspaceNavigation();

  return useMutation({
    mutationFn: async (source: RedeploySource) => {
      const res = await getUnkeyClient().deployments.createDeploymentV3({
        project: source.projectId,
        app: source.appId,
        environment: source.environmentId,
        deployment: { deploymentId: source.id },
      });
      return { deploymentId: res.data.deploymentId };
    },
    onSuccess: async (data, source) => {
      refetchDeployments();
      await collection.apps.utils.refetch();
      navigate(
        routes.projects.apps.deployment({
          workspaceSlug: workspace.slug,
          projectId: source.projectId,
          appId: source.appId,
          deploymentId: data.deploymentId,
        }),
      );
      dismissSettingsBanner();
    },
    onError: (error) => {
      toast.error("Redeploy failed", { description: getErrorMessage(error) });
    },
  });
}
