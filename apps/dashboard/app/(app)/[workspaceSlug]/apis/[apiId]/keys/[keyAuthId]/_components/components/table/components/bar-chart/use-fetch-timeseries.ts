import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useTRPC } from "@/lib/trpc/client";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import { useEffect, useMemo, useRef, useState } from "react";

import { useQuery } from "@tanstack/react-query";

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

type CacheEntry = {
  data: { timeseries: VerificationTimeseriesDataPoint[] };
  timestamp: number;
};

const timeseriesCache = new Map<string, CacheEntry>();

export const useFetchVerificationTimeseries = (keyAuthId: string, keyId: string) => {
  const trpc = useTRPC();
  // Use a ref for the initial timestamp to keep it stable
  const initialTimeRef = useRef(Date.now());
  const cacheKey = `${keyAuthId}-${keyId}`;

  // Check if we have cached data
  const cachedData = timeseriesCache.get(cacheKey);

  // State to force updates when cache changes
  const [_, setCacheVersion] = useState(0);

  // Determine if we should run the query
  const shouldFetch = !cachedData || Date.now() - cachedData.timestamp > 60000;

  // Set up query parameters - stable between renders
  const queryParams = useMemo(
    () => ({
      startTime: initialTimeRef.current - HISTORICAL_DATA_WINDOW * 3,
      endTime: initialTimeRef.current,
      keyAuthId,
      keyId,
    }),
    [keyAuthId, keyId],
  );

  // Use TRPC's useQuery with critical settings
  const {
    data,
    isLoading: trpcIsLoading,
    isError,
  } = useQuery(trpc.api.keys.usageTimeseries.queryOptions(queryParams, {
    // CRITICAL: Only enable the query if we should fetch
    enabled: shouldFetch,
    // Prevent automatic refetching
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    staleTime: Number.POSITIVE_INFINITY,
    refetchInterval: shouldFetch && queryParams.endTime >= Date.now() - 60_000 ? 10_000 : false,
  }));

  // Process the timeseries data - using cached or fresh data
  const effectiveData = data || (cachedData ? cachedData.data : undefined);

  // Process the timeseries from the effective data
  const timeseries = useMemo(() => {
    if (!effectiveData?.timeseries) {
      return [] as ProcessedTimeseriesDataPoint[];
    }

    return effectiveData.timeseries.map((ts): ProcessedTimeseriesDataPoint => {
      const result: ProcessedTimeseriesDataPoint = {
        valid: ts.y.valid,
        total: ts.y.total,
        success: ts.y.valid,
        error: ts.y.total - ts.y.valid,
      };

      // Add optional fields if they exist
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
    });
  }, [effectiveData]);

  // Update cache when we get new data
  useEffect(() => {
    if (data) {
      timeseriesCache.set(cacheKey, {
        data,
        timestamp: Date.now(),
      });
      // Force a re-render to use cached data
      setCacheVersion((prev) => prev + 1);
    }
  }, [data, cacheKey]);

  const isLoading = trpcIsLoading && !cachedData;

  return {
    timeseries,
    isLoading,
    isError,
  };
};
