import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { TIMESERIES_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { useFilters } from "../../../../hooks/use-filters";
import type { RatelimitOverviewQueryTimeseriesPayload } from "../../bar-chart/query-timeseries.schema";

export const useFetchRatelimitOverviewLatencyTimeseries = (namespaceId: string) => {
  const { filters } = useFilters();
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = useMemo(() => {
    const params: RatelimitOverviewQueryTimeseriesPayload = {
      namespaceId,
      startTime: dateNow - TIMESERIES_DATA_WINDOW,
      endTime: dateNow,
      identifiers: { filters: [] },
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "identifiers": {
          if (typeof filter.value !== "string") {
            console.error("Identifier filter value type has to be 'string'");
            return;
          }
          params.identifiers?.filters.push({
            operator: filter.operator as "is" | "contains",
            value: filter.value,
          });
          break;
        }
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'number'`);
            return;
          }
          params[filter.field] = filter.value;
          break;
        }
        case "since": {
          if (typeof filter.value !== "string") {
            console.error("Since filter value type has to be 'string'");
            return;
          }
          params.since = filter.value;
          break;
        }
      }
    });

    return params;
  }, [filters, dateNow, namespaceId]);

  const { data, isLoading, isError } =
    trpc.ratelimit.overview.logs.queryRatelimitLatencyTimeseries.useQuery(queryParams, {
      refetchInterval: queryParams.endTime ? false : 10_000,
    });

  const timeseries = data?.timeseries.map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data.granularity),
    originalTimestamp: ts.x,
    avgLatency: ts.y.avg_latency,
    p99Latency: ts.y.p99_latency,
  }));

  return {
    latencyTimeseries: timeseries,
    latencyIsLoading: isLoading,
    latencyIsError: isError,
  };
};
