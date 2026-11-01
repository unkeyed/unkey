import { useAppScope, useDeployActions } from "@/lib/deploy-data";
import type { DeployGitSource } from "@/lib/deploy-ref";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";

type DeploymentSource = { environment: string } & ({ git: DeployGitSource } | { oci: string });

export function useCreateDeployment(onDeployed: () => void) {
  const { projectId, appId } = useAppScope();
  const { refetchDeployments, refetchApps } = useDeployActions();

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
      await refetchApps();
    },
    onError(err, variables) {
      console.error("createDeployment failed", {
        projectId,
        appId,
        environment: variables.environment,
        error: err,
      });
      toast.error(getErrorMessage(err));
    },
  });
}
