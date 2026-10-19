import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { getApiRequestsTimeseries, toApiRequestsFilters } from "@/lib/api-request-logs";
import { queryKeys } from "@/lib/query-keys";
import { useQueryTime } from "@/providers/query-time-provider";
import { useQuery } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import { useFilters } from "../../../hooks/use-filters";

export const useFetchTimeseries = () => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  // The aggregates have no request id, so the chart ignores that filter
  const apiFilters = useMemo(() => {
    const { requestId: _requestId, ...timeseriesFilters } = toApiRequestsFilters(filters, {
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
    });
    return timeseriesFilters;
  }, [filters, timestamp]);

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.logs.apiRequests.timeseries(apiFilters),
    queryFn: ({ signal }) => getApiRequestsTimeseries(apiFilters, signal),
    onError(error) {
      toast.error(
        (error instanceof Error && error.message) || "Could not load logs timeseries data.",
      );
    },
  });

  const timeseries = data?.points.map((point) => ({
    displayX: formatTimestampForChart(point.time, data.granularity),
    originalTimestamp: point.time,
    success: point.success,
    warning: point.clientError,
    error: point.serverError,
    total: point.total,
  }));

  return { timeseries, isLoading, isError, bucketMs: data?.bucketMs };
};
