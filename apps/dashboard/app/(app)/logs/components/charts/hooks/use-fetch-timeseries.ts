import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { buildQueryParams } from "../../../filters.query-params";

export const useFetchTimeseries = () => {
  const { queryTime: timestamp } = useQueryTime();
  const queryParams = buildQueryParams({ timestamp });

  const { data, isLoading, isError } = trpc.logs.queryTimeseries.useQuery(queryParams, {
    refetchInterval: queryParams.endTime ? false : 10_000,
  });

  const timeseries = data?.timeseries.map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data.granularity),
    originalTimestamp: ts.x,
    ...ts.y,
  }));

  return { timeseries, isLoading, isError, granularity: data?.granularity };
};
