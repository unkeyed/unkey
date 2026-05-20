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
  const { data, isLoading, isError } = trpc.deploy.metrics.getDeploymentLatencyMetrics.useQuery(
    { deploymentId, percentile },
    { refetchInterval: 10_000 },
  );

  const timeseries =
    data?.timeseries.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [];

  return {
    currentLatency: data?.current ?? 0,
    timeseries,
    isLoading,
    isError,
  };
}
