import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";

const RUNTIME_LOGS_REFETCH_INTERVAL = 2000;
const RUNTIME_LOGS_LIMIT = 10;
const RUNTIME_LOGS_SINCE = "6h";

type UseRuntimeLogsProps = {
  projectId: string;
  deploymentId: string;
};

type UseRuntimeLogsReturn = {
  logs: RuntimeLog[];
  isLoading: boolean;
};

export function useRuntimeLogs({
  projectId,
  deploymentId,
}: UseRuntimeLogsProps): UseRuntimeLogsReturn {
  const { queryTime: timestamp } = useQueryTime();

  const { data, isLoading } = trpc.deploy.runtimeLogs.query.useQuery(
    {
      projectId,
      deploymentId,
      limit: RUNTIME_LOGS_LIMIT,
      startTime: timestamp,
      endTime: timestamp,
      since: RUNTIME_LOGS_SINCE,
      severity: null,
      message: null,
    },
    {
      refetchInterval: RUNTIME_LOGS_REFETCH_INTERVAL,
      refetchOnWindowFocus: false,
    },
  );

  return {
    logs: data?.logs ?? [],
    isLoading,
  };
}
