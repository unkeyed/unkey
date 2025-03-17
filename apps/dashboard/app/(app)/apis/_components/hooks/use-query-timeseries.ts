import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { TIMESERIES_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";
import type { VerificationQueryTimeseriesPayload } from "./query-timeseries.schema";
import { useFilters } from "./use-filters";

export const useFetchVerificationTimeseries = (keyspaceId: string | null) => {
  const [enabled, setEnabled] = useState(false);
  const { filters } = useFilters();
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = useMemo(() => {
    const params: VerificationQueryTimeseriesPayload = {
      keyspaceId: keyspaceId ?? "",
      startTime: dateNow - TIMESERIES_DATA_WINDOW * 24,
      endTime: dateNow,
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
  }, [filters, dateNow, keyspaceId]);

  useEffect(() => {
    // Implement a 2-second delay before enabling queries to prevent excessive ClickHouse load
    // during component mounting cycles. This throttling is critical when users are actively searching/filtering, to avoid
    // overwhelming the database with redundant or intermediate query states.
    setTimeout(() => setEnabled(true), 2000);
  }, []);

  const { data, isLoading, isError } = trpc.api.overview.timeseries.useQuery(queryParams, {
    refetchInterval: queryParams.endTime ? false : 10_000,
    enabled,
  });

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
