import { trpc } from "@/lib/trpc/client";
import { formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { type TimeseriesData, downsample } from "./downsample";

type UseDeploymentMemoryResult = {
  memoryPercent: number;
  memoryDisplay: { value: string; unit: string };
  allocatedBytes: number;
  timeseries: TimeseriesData[];
  isLoading: boolean;
  isError: boolean;
};

export function useDeploymentMemory(deploymentId: string): UseDeploymentMemoryResult {
  const {
    data: timeseriesData,
    isLoading: isTimeseriesLoading,
    isError: isTimeseriesError,
  } = trpc.deploy.metrics.getDeploymentMemoryTimeseries.useQuery(
    { resourceId: deploymentId, window: "6h" },
    { refetchInterval: 10_000 },
  );

  const {
    data: summary,
    isLoading: isSummaryLoading,
    isError: isSummaryError,
  } = trpc.deploy.metrics.getDeploymentResourceSummary.useQuery(
    { resourceId: deploymentId },
    { refetchInterval: 10_000 },
  );

  const timeseries = downsample(
    timeseriesData?.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [],
  );

  const allocated = summary?.memory_allocated_bytes ?? 0;
  const current = summary?.current_memory_bytes ?? 0;
  const memoryPercent = allocated > 0 ? Math.round((current / allocated) * 100) : 0;
  const memoryDisplay = formatMemoryParts(Math.round(current / (1024 * 1024)));

  return {
    memoryPercent,
    memoryDisplay,
    allocatedBytes: allocated,
    timeseries,
    isLoading: isTimeseriesLoading || isSummaryLoading,
    isError: isTimeseriesError || isSummaryError,
  };
}
