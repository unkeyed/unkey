import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { trpc } from "@/lib/trpc/client";
import { useAppId, useProjectData } from "../../../data-provider";

export function useActiveBranches() {
  const { projectId } = useProjectData();
  const appId = useAppId();

  const query = trpc.deploy.deployment.listActiveBranches.useQuery(
    { projectId, appId },
    {
      refetchInterval: (data) =>
        data?.some((deployment) => isDeploymentInFlight(deployment.status)) ? 5_000 : false,
    },
  );

  return {
    branches: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  };
}
