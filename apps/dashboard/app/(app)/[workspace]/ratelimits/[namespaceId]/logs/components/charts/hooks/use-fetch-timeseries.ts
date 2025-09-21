import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useMemo } from "react";
import { useFilters } from "../../../hooks/use-filters";
import type { RatelimitQueryTimeseriesPayload } from "../query-timeseries.schema";

export const useFetchRatelimitTimeseries = (namespaceId: string) => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: RatelimitQueryTimeseriesPayload = {
      namespaceId,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
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
  }, [filters, namespaceId, timestamp]);

  const { data, isLoading, isError } = trpc.ratelimit.logs.queryRatelimitTimeseries.useQuery(
    queryParams,
    {
      refetchInterval: queryParams.endTime ? false : 10_000,
    },
  );

  const timeseries = data?.timeseries.map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data.granularity),
    originalTimestamp: ts.x,
    success: ts.y.passed,
    error: ts.y.total - ts.y.passed,
    total: ts.y.total,
  }));

  return { timeseries, isLoading, isError, granularity: data?.granularity };
};
