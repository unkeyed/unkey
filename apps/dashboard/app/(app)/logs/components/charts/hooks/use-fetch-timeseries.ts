import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { TIMESERIES_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import type { z } from "zod";
import { useFilters } from "../../../hooks/use-filters";
import type { queryTimeseriesPayload } from "../query-timeseries.schema";

export const useFetchTimeseries = () => {
  const { filters } = useFilters();

  const dateNow = useMemo(() => Date.now(), []);
  const queryParams = useMemo(() => {
    const params: z.infer<typeof queryTimeseriesPayload> = {
      startTime: dateNow - TIMESERIES_DATA_WINDOW,
      endTime: dateNow,
      host: { filters: [] },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "status": {
          params.status?.filters.push({
            operator: "is",
            value: Number.parseInt(filter.value as string),
          });
          break;
        }

        case "methods": {
          if (typeof filter.value !== "string") {
            console.error("Method filter value type has to be 'string'");
            return;
          }
          params.method?.filters.push({
            operator: "is",
            value: filter.value,
          });
          break;
        }

        case "paths": {
          if (typeof filter.value !== "string") {
            console.error("Path filter value type has to be 'string'");
            return;
          }
          params.path?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "host": {
          if (typeof filter.value !== "string") {
            console.error("Host filter value type has to be 'string'");
            return;
          }
          params.host?.filters.push({
            operator: "is",
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
  }, [filters, dateNow]);

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
