import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProjectData } from "../../../../../../data-provider";
import { useDeployment } from "../../../../layout-provider";

export function useDeploymentSentinelLogsQuery() {
  const { deploymentId } = useDeployment();
  const { projectId } = useProjectData();

  const deployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) => eq(deployment.projectId, projectId))
        .where(({ deployment }) => eq(deployment.id, deploymentId)),
    [projectId, deploymentId],
  );

  const environmentId = deployment.data.at(0)?.environmentId ?? "";

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
