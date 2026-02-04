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
  const {
    data: currentRpsData,
    isLoading: isCurrentLoading,
    isError: isCurrentError,
  } = trpc.deploy.metrics.getDeploymentRps.useQuery(
    {
      deploymentId,
    },
    {
      refetchInterval: 10_000,
    },
  );

  const {
    data: timeseriesData,
    isLoading: isTimeseriesLoading,
    isError: isTimeseriesError,
  } = trpc.deploy.metrics.getDeploymentRpsTimeseries.useQuery(
    {
      deploymentId,
    },
    {
      refetchInterval: 10_000,
    },
  );

  const timeseries =
    timeseriesData?.map((d) => ({
      originalTimestamp: d.x,
      y: d.y,
      total: d.y,
    })) ?? [];

  return {
    currentRps: currentRpsData?.avg_rps ?? 0,
    timeseries,
    isLoading: isCurrentLoading || isTimeseriesLoading,
    isError: isCurrentError || isTimeseriesError,
  };
}
