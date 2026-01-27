import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useParams } from "next/navigation";
import { useProject } from "../../../layout-provider";

const REFETCH_INTERVAL_MS = 5000;
const LAST_HOUR_MS = 1000 * 60 * 60;

export function useDeploymentSentinelLogsQuery() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;
  const { projectId } = useProject();
  const { queryTime: timestamp } = useQueryTime();

  const { data, isLoading, error } = trpc.deploy.sentinelLogs.query.useQuery(
    {
      projectId,
      deploymentId,
      limit: 50,
      startTime: timestamp - LAST_HOUR_MS,
      endTime: timestamp,
    },
    {
      refetchInterval: REFETCH_INTERVAL_MS,
    },
  );

  return {
    logs: data ?? [],
    isLoading,
    error,
  };
}
