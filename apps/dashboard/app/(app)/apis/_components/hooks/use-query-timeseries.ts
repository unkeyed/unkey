import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { TIMESERIES_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import type { VerificationQueryTimeseriesPayload } from "./query-timeseries.schema";
import { useFilters } from "./use-filters";

export const useFetchVerificationTimeseries = (keyspaceId: string) => {
  const { filters } = useFilters();
  const dateNow = useMemo(() => Date.now(), []);

  const queryParams = useMemo(() => {
    const params: VerificationQueryTimeseriesPayload = {
      keyspaceId,
      startTime: dateNow - TIMESERIES_DATA_WINDOW,
      endTime: dateNow,
      since: "",
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(
              `${filter.field} filter value type has to be 'number'`
            );
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

  const { data, isLoading, isError } =
    trpc.api.logs.queryVerificationTimeseries.useQuery(queryParams, {
      refetchInterval: queryParams.endTime ? false : 10_000,
    });

  const timeseries = data?.timeseries.map((ts) => ({
    displayX: formatTimestampForChart(ts.x, data.granularity),
    originalTimestamp: ts.x,
    valid: ts.y.valid,
    insufficient_permissions: ts.y.insufficient_permissions,
    rate_limited: ts.y.rate_limited,
    forbidden: ts.y.forbidden,
    disabled: ts.y.disabled,
    expired: ts.y.expired,
    usage_exceeded: ts.y.usage_exceeded,
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
