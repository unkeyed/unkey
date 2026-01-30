import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
import { useProject } from "../../../../../../layout-provider";

export function useDeploymentSentinelLogsQuery() {
  const params = useParams();
  const deploymentId = (params?.deploymentId as string) ?? "";
  const { projectId, collections } = useProject();

  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [deploymentId],
  );

  const environmentId = deployment.data.at(0)?.environmentId ?? "";

  const { data, isLoading, error } = trpc.deploy.sentinelLogs.query.useQuery(
    {
      projectId,
      environmentId,
      deploymentId,
      limit: 50,
    },
    {
      keepPreviousData: true,
      enabled: Boolean(environmentId) && Boolean(deploymentId),
      refetchInterval: 5000,
    },
  );

  return {
    logs: data ?? [],
    isLoading,
    error,
  };
}
