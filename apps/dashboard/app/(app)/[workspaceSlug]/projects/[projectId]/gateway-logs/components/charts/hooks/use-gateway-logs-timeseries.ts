import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import type { TimeseriesRequestSchema } from "@/lib/schemas/logs.schema";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useMemo } from "react";
import { EXCLUDED_HOSTS } from "../../../constants";
import { useGatewayLogsFilters } from "../../../hooks/use-gateway-logs-filters";

import { useQuery } from "@tanstack/react-query";

// Constants
const FILTER_FIELD_MAPPING = {
  status: "status",
  methods: "method",
  paths: "path",
  host: "host",
} as const;

const TIME_FIELDS = ["startTime", "endTime", "since"] as const;

export const useGatewayLogsTimeseries = () => {
  const trpc = useTRPC();
  const { filters } = useGatewayLogsFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: TimeseriesRequestSchema = {
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      host: { filters: [], exclude: EXCLUDED_HOSTS },
      method: { filters: [] },
      path: { filters: [] },
      status: { filters: [] },
      since: "",
    };

    filters.forEach((filter) => {
      const paramKey = FILTER_FIELD_MAPPING[filter.field as keyof typeof FILTER_FIELD_MAPPING];

      if (paramKey && params[paramKey as keyof typeof params]) {
        switch (filter.field) {
          case "status": {
            const statusValue = Number.parseInt(filter.value as string);
            if (Number.isNaN(statusValue)) {
              console.error("Status filter value must be a valid number");
              return;
            }
            params.status?.filters.push({
              operator: "is",
              value: statusValue,
            });
            break;
          }

          case "methods":
          case "host": {
            if (typeof filter.value !== "string") {
              console.error(`${filter.field} filter value must be a string`);
              return;
            }
            const targetParam = params[paramKey as keyof typeof params] as {
              filters: Array<{ operator: string; value: string }>;
            };
            targetParam.filters.push({
              operator: "is",
              value: filter.value,
            });
            break;
          }

          case "paths": {
            if (typeof filter.value !== "string") {
              console.error("Path filter value must be a string");
              return;
            }
            params.path?.filters.push({
              operator: filter.operator,
              value: filter.value,
            });
            break;
          }
        }
      } else if (TIME_FIELDS.includes(filter.field as (typeof TIME_FIELDS)[number])) {
        switch (filter.field) {
          case "startTime":
          case "endTime": {
            if (typeof filter.value !== "number") {
              console.error(`${filter.field} filter value must be a number`);
              return;
            }
            params[filter.field] = filter.value;
            break;
          }
          case "since": {
            if (typeof filter.value !== "string") {
              console.error("Since filter value must be a string");
              return;
            }
            params.since = filter.value;
            break;
          }
        }
      }
    });

    return params;
  }, [filters, timestamp]);

  const { data, isLoading, isError } = useQuery(trpc.logs.queryTimeseries.queryOptions(queryParams, {
    refetchInterval: queryParams.endTime ? false : 10_000,
    trpc: {
      context: {
        skipBatch: true,
      },
    },
  }));

  const timeseries = data?.timeseries.map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data.granularity),
    originalTimestamp: ts.x,
    ...ts.y,
  }));

  return {
    timeseries,
    isLoading,
    isError,
    granularity: data?.granularity,
  };
};
