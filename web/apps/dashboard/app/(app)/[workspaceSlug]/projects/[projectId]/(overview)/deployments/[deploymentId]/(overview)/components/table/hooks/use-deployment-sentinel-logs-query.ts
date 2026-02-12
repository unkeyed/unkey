import { trpc } from "@/lib/trpc/client";
import { useProjectData } from "../../../../../../data-provider";
import { useDeployment } from "../../../../layout-provider";

export function useDeploymentSentinelLogsQuery() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById, projectId } = useProjectData();

  const deployment = getDeploymentById(deploymentId);
  const environmentId = deployment?.environmentId ?? "";

  const { data, isLoading, error } = trpc.deploy.sentinelLogs.query.useInfiniteQuery(
    {
      projectId,
      deploymentId,
      environmentId,
      limit: 50,
      since: "6h",
      statusCodes: null,
      methods: null,
      paths: null,
    },
    {
      enabled: Boolean(environmentId) && Boolean(deploymentId),
      refetchInterval: 5000,
      getNextPageParam: (lastPage) => (lastPage.hasMore ? lastPage.nextCursor : undefined),
    },
  );

  return {
    logs: data?.pages[0]?.logs ?? [],
    isLoading,
    error,
  };
}
