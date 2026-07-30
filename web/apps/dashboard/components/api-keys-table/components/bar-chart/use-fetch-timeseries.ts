import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import { useMemo, useState } from "react";

export type ProcessedTimeseriesDataPoint = {
  valid: number;
  total: number;
  success: number;
  error: number;
  rate_limited?: number;
  insufficient_permissions?: number;
  forbidden?: number;
  disabled?: number;
  expired?: number;
  usage_exceeded?: number;
};

const EMPTY_TIMESERIES: ProcessedTimeseriesDataPoint[] = [];

// Both cells in a row call this with the same key, and the window is bucketed to
// the minute, so they land on one query key and React Query serves them from a
// single cache entry instead of fetching the same data twice.
const WINDOW_BUCKET_MS = 60_000;

// Runs once per key row from both the usage column and the status cell. It used
// to keep a module-level Map, write to it from an effect, and force a re-render
// through a throwaway `useState` counter so the next render read the Map back.
// That made `enabled` and `refetchInterval` — both derived from `Date.now()` and
// from that Map — differ between renders, so React Query republished the
// observer result on every pass and `useSyncExternalStore` never settled
// (ENG-3091). Everything the query depends on is now fixed at mount.
export const useFetchVerificationTimeseries = (keyAuthId: string, keyId: string) => {
  const [anchor] = useState(() => Math.floor(Date.now() / WINDOW_BUCKET_MS) * WINDOW_BUCKET_MS);

  const queryParams = useMemo(
    () => ({
      startTime: anchor - HISTORICAL_DATA_WINDOW * 3,
      endTime: anchor,
      keyAuthId,
      keyId,
    }),
    [anchor, keyAuthId, keyId],
  );

  const { data, isLoading, isError } = trpc.api.keys.usageTimeseries.useQuery(queryParams, {
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnWindowFocus: false,
  });

  const timeseries = useMemo(() => {
    if (!data?.timeseries) {
      return EMPTY_TIMESERIES;
    }

    return data.timeseries.map(
      (ts: VerificationTimeseriesDataPoint): ProcessedTimeseriesDataPoint => {
        const result: ProcessedTimeseriesDataPoint = {
          valid: ts.y.valid,
          total: ts.y.total,
          success: ts.y.valid,
          error: ts.y.total - ts.y.valid,
        };

        if (ts.y.rate_limited_count !== undefined) {
          result.rate_limited = ts.y.rate_limited_count;
        }
        if (ts.y.insufficient_permissions_count !== undefined) {
          result.insufficient_permissions = ts.y.insufficient_permissions_count;
        }
        if (ts.y.forbidden_count !== undefined) {
          result.forbidden = ts.y.forbidden_count;
        }
        if (ts.y.disabled_count !== undefined) {
          result.disabled = ts.y.disabled_count;
        }
        if (ts.y.expired_count !== undefined) {
          result.expired = ts.y.expired_count;
        }
        if (ts.y.usage_exceeded_count !== undefined) {
          result.usage_exceeded = ts.y.usage_exceeded_count;
        }

        return result;
      },
    );
  }, [data]);

  return {
    timeseries,
    isLoading,
    isError,
  };
};
