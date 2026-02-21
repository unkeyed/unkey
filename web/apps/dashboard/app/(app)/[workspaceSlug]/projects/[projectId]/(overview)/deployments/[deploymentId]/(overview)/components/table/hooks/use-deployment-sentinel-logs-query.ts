import { trpc } from "@/lib/trpc/client";
import { useDeployment } from "../../../../layout-provider";

export function useDeploymentSentinelLogsQuery() {
  const { deployment } = useDeployment();

  const { data, isLoading, error } = trpc.deploy.sentinelLogs.query.useInfiniteQuery(
    {
      projectId: deployment.projectId,
      deploymentId: deployment.id,
      environmentId: deployment.environmentId,
      limit: 50,
      since: "6h",
      statusCodes: null,
      methods: null,
      paths: null,
    },
    {
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
