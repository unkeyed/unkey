import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useEffect, useMemo, useState } from "react";
import type { VerificationQueryTimeseriesPayload } from "./query-timeseries.schema";
import { useFilters } from "./use-filters";

import { useQuery } from "@tanstack/react-query";

export const useFetchVerificationTimeseries = (keyspaceId: string | null) => {
  const trpc = useTRPC();
  const [enabled, setEnabled] = useState(false);
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: VerificationQueryTimeseriesPayload = {
      keyspaceId: keyspaceId ?? "",
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
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
  }, [filters, timestamp, keyspaceId]);

  useEffect(() => {
    // Implement a 2-second delay before enabling queries to prevent excessive ClickHouse load
    // during component mounting cycles. This throttling is critical when users are actively searching/filtering, to avoid
    // overwhelming the database with redundant or intermediate query states.
    setTimeout(() => setEnabled(true), 2000);
  }, []);

  const { data, isLoading, isError } = useQuery(
    trpc.api.overview.timeseries.queryOptions(queryParams, {
      refetchInterval: queryParams.endTime ? false : 10_000,
      enabled,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    }),
  );

  const timeseries = (data?.timeseries ?? []).map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data?.granularity ?? "per12Hours"),
    originalTimestamp: ts.x,
    valid: ts.y.valid,
    total: ts.y.total,
    success: ts.y.valid,
    error: ts.y.total - ts.y.valid,
  }));

  return {
    timeseries,
    isLoading,
    isError,
    granularity: data?.granularity,
  };
};
