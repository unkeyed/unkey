import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import type { QueryLogsPayload } from "./filters.schema";
import { useFilters } from "./hooks/use-filters";

type BuildQueryParamsOptions = {
  timestamp: number;
  limit?: number;
};

export function buildQueryParams({ timestamp, limit }: BuildQueryParamsOptions): QueryLogsPayload {
  const { filters } = useFilters();
  const params: QueryLogsPayload = {
    startTime: timestamp - HISTORICAL_DATA_WINDOW,
    endTime: timestamp,
    host: { filters: [] },
    requestId: { filters: [] },
    methods: { filters: [] },
    paths: { filters: [] },
    status: { filters: [] },
    since: "",
    // Timeseries queries will ignore this prop
    limit: limit ?? 0,
  };

  for (const filter of filters) {
    switch (filter.field) {
      case "status": {
        const statusValue = Number.parseInt(filter.value as string);
        if (Number.isNaN(statusValue)) {
          throw new Error(`Invalid status filter value: ${filter.value}`);
        }
        params.status?.filters.push({
          operator: "is",
          value: statusValue,
        });
        break;
      }

      case "methods":
      case "paths":
      case "host":
      case "requestId": {
        if (typeof filter.value !== "string") {
          throw new Error(`${filter.field} filter value must be a string`);
        }
        params[filter.field]?.filters.push({
          operator: filter.operator,
          value: filter.value,
        });
        break;
      }

      case "startTime":
      case "endTime": {
        if (typeof filter.value !== "number") {
          throw new Error(`${filter.field} filter value must be a number`);
        }
        params[filter.field] = filter.value;
        break;
      }

      case "since": {
        if (typeof filter.value !== "string") {
          throw new Error("Since filter value must be a string");
        }
        params.since = filter.value;
        break;
      }

      default: {
        const _exhaustive: unknown = filter.field;
        throw new Error(`Unknown filter field: ${_exhaustive}`);
      }
    }
  }

  return params;
}
