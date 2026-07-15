import { trpc } from "@/lib/trpc/client";

type TimeseriesData = {
  originalTimestamp: number;
  y: number;
  total: number;
};

type UseDeploymentRpsResult = {
  currentRps: number;
  timeseries: TimeseriesData[];
  isLoading: boolean;
  isError: boolean;
};

export function useDeploymentRps(deploymentId: string): UseDeploymentRpsResult {
  const { data, isLoading, isError } = trpc.deploy.metrics.getDeploymentRpsMetrics.useQuery(
    { deploymentId },
    { refetchInterval: 10_000 },
  );

  const timeseries =
    data?.timeseries.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [];

  return {
    currentRps: data?.current ?? 0,
    timeseries,
    isLoading,
    isError,
  };
}
