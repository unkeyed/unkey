import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useEffect, useMemo, useState } from "react";
import type { VerificationQueryTimeseriesPayload } from "./query-timeseries.schema";

export type VerificationTimeseriesPoint = {
  displayX: string;
  originalTimestamp: number;
  valid: number;
  total: number;
  success: number;
  error: number;
};

export const useFetchVerificationTimeseries = (keyspaceId: string | null) => {
  const [enabled, setEnabled] = useState(false);
  const { queryTime: timestamp } = useQueryTime();

  // The list page has no time filters; every card chart uses a fixed 12h window.
  const queryParams = useMemo((): VerificationQueryTimeseriesPayload => {
    return {
      keyspaceId: keyspaceId ?? "",
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      since: "",
    };
  }, [timestamp, keyspaceId]);

  useEffect(() => {
    // Implement a 2-second delay before enabling queries to prevent excessive ClickHouse load
    // during component mounting cycles. This throttling is critical when users are actively searching/filtering, to avoid
    // overwhelming the database with redundant or intermediate query states.
    setTimeout(() => setEnabled(true), 2000);
  }, []);

  const { data, isLoading, isError } = trpc.api.overview.timeseries.useQuery(queryParams, {
    refetchInterval: false,
    enabled,
    trpc: {
      context: {
        skipBatch: true,
      },
    },
  });

  const timeseries = data?.timeseries?.map(
    (ts): VerificationTimeseriesPoint => ({
      displayX: formatTimestampForChart(ts.x, data.granularity ?? "per12Hours"),
      originalTimestamp: ts.x,
      valid: ts.y.valid,
      total: ts.y.total,
      success: ts.y.valid,
      error: ts.y.total - ts.y.valid,
    }),
  );

  return {
    timeseries,
    isLoading,
    isError,
    granularity: data?.granularity,
  };
};
