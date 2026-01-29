import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
import { useEffect } from "react";
import { useProject } from "../../../../../../layout-provider";

const REFETCH_INTERVAL_MS = 5000;
const LAST_HOUR_MS = 1000 * 60 * 60;

export function useDeploymentSentinelLogsQuery() {
  const params = useParams();
  const deploymentId = (params?.deploymentId as string) ?? "";
  const { projectId, collections } = useProject();
  const { queryTime: timestamp, refreshQueryTime } = useQueryTime();

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
      startTime: timestamp - LAST_HOUR_MS,
      endTime: timestamp,
    },
    { keepPreviousData: true, enabled: Boolean(environmentId) && Boolean(deploymentId) },
  );

  useEffect(() => {
    const timer = setInterval(() => {
      refreshQueryTime();
    }, REFETCH_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [refreshQueryTime]);

  return {
    logs: data ?? [],
    isLoading,
    error,
  };
}
