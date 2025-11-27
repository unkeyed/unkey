import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import type { TimeseriesRequestSchema } from "@/lib/schemas/logs.schema";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useMemo } from "react";
import { useFilters } from "../../../hooks/use-filters";

import { useQuery } from "@tanstack/react-query";

export const useFetchTimeseries = () => {
  const trpc = useTRPC();
  const { filters } = useFilters();

  const { queryTime: timestamp } = useQueryTime();
  const queryParams = useMemo(() => {
    const params: TimeseriesRequestSchema = {
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
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
          params.host?.filters?.push({
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
  }, [filters, timestamp]);

  const { data, isLoading, isError } = useQuery(
    trpc.logs.queryTimeseries.queryOptions(queryParams, {
      refetchInterval: queryParams.endTime ? false : 10_000,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    }),
  );

  const timeseries = data?.timeseries.map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data.granularity),
    originalTimestamp: ts.x,
    ...ts.y,
  }));

  return { timeseries, isLoading, isError, granularity: data?.granularity };
};
