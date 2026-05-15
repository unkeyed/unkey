import { trpc } from "@/lib/trpc/client";
import { type TimeseriesData, downsample } from "./downsample";

type UseDeploymentCpuResult = {
  cpuPercent: number;
  allocatedMillicores: number;
  timeseries: TimeseriesData[];
  isLoading: boolean;
  isError: boolean;
};

export function useDeploymentCpu(deploymentId: string): UseDeploymentCpuResult {
  const {
    data: timeseriesData,
    isLoading: isTimeseriesLoading,
    isError: isTimeseriesError,
  } = trpc.deploy.metrics.getDeploymentCpuTimeseries.useQuery(
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

  const allocated = summary?.cpu_allocated_millicores ?? 0;
  const current = summary?.current_cpu_millicores ?? 0;
  const cpuPercent = allocated > 0 ? Math.round((current / allocated) * 100) : 0;

  return {
    cpuPercent,
    allocatedMillicores: allocated,
    timeseries,
    isLoading: isTimeseriesLoading || isSummaryLoading,
    isError: isTimeseriesError || isSummaryError,
  };
}
