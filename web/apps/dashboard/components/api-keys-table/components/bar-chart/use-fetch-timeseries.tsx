"use client";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { type ReactNode, createContext, useContext, useMemo, useState } from "react";
import {
  EMPTY_TIMESERIES,
  type ProcessedTimeseriesDataPoint,
  buildTimeseriesByKey,
} from "./timeseries-model";

export type { ProcessedTimeseriesDataPoint } from "./timeseries-model";

// Both cells in a row read the same key, and the whole page shares one query
// keyed on the visible key ids and a window bucketed to the minute, so React
// Query fetches the usage timeseries for every visible key in a single request.
const WINDOW_BUCKET_MS = 60_000;

type KeysUsageTimeseries = {
  get: (keyId: string) => ProcessedTimeseriesDataPoint[];
  isLoading: boolean;
  isError: boolean;
};

const KeysUsageTimeseriesContext = createContext<KeysUsageTimeseries | null>(null);

export const KeysUsageTimeseriesProvider = ({
  keyAuthId,
  keyIds,
  children,
}: {
  keyAuthId: string;
  keyIds: string[];
  children: ReactNode;
}) => {
  const [anchor] = useState(() => Math.floor(Date.now() / WINDOW_BUCKET_MS) * WINDOW_BUCKET_MS);
  const startTime = anchor - HISTORICAL_DATA_WINDOW * 3;

  const sortedKeyIds = useMemo(() => [...keyIds].sort(), [keyIds]);

  const { data, isLoading, isError } = trpc.api.keys.usageTimeseries.useQuery(
    { startTime, endTime: anchor, keyAuthId, keyIds: sortedKeyIds },
    {
      enabled: sortedKeyIds.length > 0,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnWindowFocus: false,
    },
  );

  const value = useMemo<KeysUsageTimeseries>(() => {
    const byKeyId = buildTimeseriesByKey(data?.series, sortedKeyIds, startTime, anchor);

    return {
      get: (keyId: string) => byKeyId.get(keyId) ?? EMPTY_TIMESERIES,
      isLoading,
      isError,
    };
  }, [data, sortedKeyIds, startTime, anchor, isLoading, isError]);

  return (
    <KeysUsageTimeseriesContext.Provider value={value}>
      {children}
    </KeysUsageTimeseriesContext.Provider>
  );
};

export const useFetchVerificationTimeseries = (keyId: string) => {
  const context = useContext(KeysUsageTimeseriesContext);

  if (!context) {
    throw new Error(
      "useFetchVerificationTimeseries must be used inside KeysUsageTimeseriesProvider",
    );
  }

  return {
    timeseries: context.get(keyId),
    isLoading: context.isLoading,
    isError: context.isError,
  };
};
