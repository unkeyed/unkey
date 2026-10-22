import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { toApiRequestsFilters } from "@/lib/api-request-logs";
import { queryKeys } from "@/lib/query-keys";
import {
  TIMESERIES_GRANULARITIES,
  type TimeseriesGranularity,
} from "@/lib/trpc/routers/utils/granularity";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useQuery } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";

const LIVE_REFRESH_INTERVAL_MS = 10_000;

export const useFetchTimeseries = () => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();
  const { isLive } = useLogsContext();

  // The aggregates have no request id, so the chart ignores that filter
  const apiFilters = useMemo(() => {
    const { requestId: _requestId, ...timeseriesFilters } = toApiRequestsFilters(filters, {
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
    });
    return timeseriesFilters;
  }, [filters, timestamp]);

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.logs.apiRequests.metrics(apiFilters, isLive),
    queryFn: async ({ signal }) => {
      // Live mode keeps the start and grows the window to now, like the table
      const gte = apiFilters.time?.gte;
      const time = isLive && gte !== undefined ? { gte, lte: Date.now() } : apiFilters.time;
      const { data } = await getUnkeyClient().logs.getApiRequestMetrics(
        { filters: { ...apiFilters, time } },
        { signal },
      );
      return data;
    },
    select: (points) => {
      // Points have no gaps, so two of them give the bucket width. A single
      // point spans the whole window
      const { gte, lte } = apiFilters.time ?? {};
      const bucketMs =
        points.length >= 2
          ? points[1].time - points[0].time
          : gte !== undefined && lte !== undefined
            ? lte - gte + 1
            : undefined;
      // The widest granularity that fits in a bucket, ordered narrowest first
      const granularity =
        Object.keys(TIMESERIES_GRANULARITIES)
          .filter(isTimeseriesGranularity)
          .findLast((g) => bucketMs !== undefined && TIMESERIES_GRANULARITIES[g].ms <= bucketMs) ??
        "perMinute";
      return {
        bucketMs,
        granularity,
        timeseries: points.map((point) => ({
          ...point,
          displayX: formatTimestampForChart(point.time, granularity),
          originalTimestamp: point.time,
        })),
      };
    },
    refetchInterval: isLive ? LIVE_REFRESH_INTERVAL_MS : false,
    onError(error) {
      toast.error(
        (error instanceof Error && error.message) || "Could not load logs timeseries data.",
      );
    },
  });

  return { ...data, isLoading, isError };
};

function isTimeseriesGranularity(value: string): value is TimeseriesGranularity {
  return value in TIMESERIES_GRANULARITIES;
}
