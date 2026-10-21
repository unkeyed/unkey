import { collection } from "@/lib/collections";
import type { DeployGitSource } from "@/lib/deploy-ref";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useAppId, useProjectData } from "../data-provider";

type DeploymentSource = { environment: string } & ({ git: DeployGitSource } | { oci: string });

export function useCreateDeployment(onDeployed: () => void) {
  const { projectId, refetchDeployments } = useProjectData();
  const appId = useAppId();

  return useMutation({
    mutationFn: async (source: DeploymentSource) => {
      await getUnkeyClient().deployments.createDeploymentV3({
        project: projectId,
        app: appId,
        environment: source.environment,
        ...("oci" in source ? { oci: { image: source.oci } } : { git: source.git }),
      });
    },
    async onSuccess() {
      toast.success("Deployment has been created");
      onDeployed();
      refetchDeployments();
      await collection.apps.utils.refetch();
    },
    onError(err) {
      console.error(err);
      toast.error(getErrorMessage(err));
    },
  });
}
