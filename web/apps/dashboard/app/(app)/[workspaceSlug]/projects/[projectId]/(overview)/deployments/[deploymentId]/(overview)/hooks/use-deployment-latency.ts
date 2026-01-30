import { trpc } from "@/lib/trpc/client";

type TimeseriesData = {
  originalTimestamp: number;
  y: number;
  total: number;
};

type LatencyPercentile = "p50" | "p75" | "p90" | "p95" | "p99";

type UseDeploymentLatencyResult = {
  currentLatency: number;
  timeseries: TimeseriesData[];
  isLoading: boolean;
  isError: boolean;
};

export function useDeploymentLatency(
  deploymentId: string,
  percentile: LatencyPercentile,
): UseDeploymentLatencyResult {
  const {
    data: currentLatencyData,
    isLoading: isCurrentLoading,
    isError: isCurrentError,
  } = trpc.deploy.metrics.getDeploymentLatency.useQuery({
    deploymentId,
    percentile,
  });

  const {
    data: timeseriesData,
    isLoading: isTimeseriesLoading,
    isError: isTimeseriesError,
  } = trpc.deploy.metrics.getDeploymentLatencyTimeseries.useQuery({
    deploymentId,
    percentile,
  });

  const timeseries =
    timeseriesData?.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [];

  return {
    currentLatency: currentLatencyData?.latency ?? 0,
    timeseries,
    isLoading: isCurrentLoading || isTimeseriesLoading,
    isError: isCurrentError || isTimeseriesError,
  };
}
