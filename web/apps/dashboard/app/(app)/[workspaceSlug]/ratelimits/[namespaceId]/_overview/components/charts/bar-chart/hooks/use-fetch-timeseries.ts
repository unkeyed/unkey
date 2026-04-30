import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useMemo } from "react";
import { useFilters } from "../../../../hooks/use-filters";
import type { RatelimitOverviewQueryTimeseriesPayload } from "../query-timeseries.schema";

export const useFetchRatelimitOverviewTimeseries = (namespaceId: string) => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: RatelimitOverviewQueryTimeseriesPayload = {
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
  }, [filters, timestamp, namespaceId]);

  const { data, isLoading, isError } = trpc.ratelimit.logs.queryRatelimitTimeseries.useQuery(
    queryParams,
    {
      refetchInterval: queryParams.endTime ? false : 10_000,
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    },
  );

  // Status filter is applied client-side: the timeseries query returns
  // both passed and total counts in the same row, so masking out the
  // un-selected series here keeps the charts in sync with the table
  // without re-querying ClickHouse.
  const statusValues = filters
    .filter((f) => f.field === "status")
    .map((f) => f.value)
    .filter((v): v is "passed" | "blocked" => v === "passed" || v === "blocked");
  const showPassed = statusValues.length === 0 || statusValues.includes("passed");
  const showBlocked = statusValues.length === 0 || statusValues.includes("blocked");

  const timeseries = data?.timeseries.map((ts) => {
    const success = showPassed ? ts.y.passed : 0;
    const error = showBlocked ? ts.y.total - ts.y.passed : 0;
    return {
      displayX: formatTimestampForChart(ts.x, data.granularity),
      originalTimestamp: ts.x,
      success,
      error,
      total: success + error,
    };
  });

  // Tokens series mirrors the request series but on `passed_tokens` and
  // the implied `blocked_tokens = total_tokens - passed_tokens`. Sharing
  // the underlying tRPC call keeps the page to one round trip.
  const tokensTimeseries = data?.timeseries.map((ts) => {
    const success = showPassed ? ts.y.passed_tokens : 0;
    const error = showBlocked ? ts.y.total_tokens - ts.y.passed_tokens : 0;
    return {
      displayX: formatTimestampForChart(ts.x, data.granularity),
      originalTimestamp: ts.x,
      success,
      error,
      total: success + error,
    };
  });

  return {
    timeseries,
    tokensTimeseries,
    isLoading,
    isError,
    granularity: data?.granularity,
  };
};
