import { trpc } from "@/lib/trpc/client";
import type { TimeseriesGranularity } from "@/lib/trpc/routers/logs/query-timeseries/utils";
import { addMinutes, format } from "date-fns";
import { useMemo } from "react";
import type { z } from "zod";
import { useFilters } from "../../../hooks/use-filters";
import type { queryTimeseriesPayload } from "../query-timeseries.schema";

// Duration in milliseconds for historical data fetch window (1 hours)
const TIMESERIES_DATA_WINDOW = 60 * 60 * 1000;

const formatTimestamp = (value: string | number, granularity: TimeseriesGranularity) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);

  switch (granularity) {
    case "perMinute":
      return format(localDate, "HH:mm:ss");
    case "perHour":
      return format(localDate, "MMM d, HH:mm");
    case "perDay":
      return format(localDate, "MMM d");
    default:
      return format(localDate, "Pp");
  }
};

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
    displayX: formatTimestamp(ts.x, data.granularity),
    originalTimestamp: ts.x,
    ...ts.y,
  }));

  return { timeseries, isLoading, isError };
};
