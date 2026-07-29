"use client";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import { type PropsWithChildren, createContext, useContext, useMemo, useState } from "react";

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

export type KeyTimeseries = {
  timeseries: ProcessedTimeseriesDataPoint[];
  isLoading: boolean;
  isError: boolean;
};

const EMPTY_TIMESERIES: ProcessedTimeseriesDataPoint[] = [];
const PENDING: KeyTimeseries = {
  timeseries: EMPTY_TIMESERIES,
  isLoading: true,
  isError: false,
};

const KeyTimeseriesContext = createContext<Map<string, KeyTimeseries> | null>(null);

const process = (points: VerificationTimeseriesDataPoint[]): ProcessedTimeseriesDataPoint[] =>
  points.map((ts) => {
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
  });

// The usage column and the status cell both need this data, so fetching it per
// cell put two query observers on every row. Fetching once here for the whole
// page keeps it to one per key, and leaves the cells as plain consumers with no
// observers of their own — the fan-out that turned an unstable observer result
// into a nested-update storm (ENG-3091).
export const KeyTimeseriesProvider = ({
  keyAuthId,
  keyIds,
  children,
}: PropsWithChildren<{ keyAuthId: string; keyIds: string[] }>) => {
  // Anchored once per mount, so revisiting the page gets a fresh window while
  // the query key stays stable across renders. The app-wide `queryTime` is not
  // usable here: it is seeded when the bundle loads and only advanced by the
  // refresh controls on other routes, so this table would keep showing the
  // window from whenever the tab was opened.
  const [anchor] = useState(() => Date.now());

  const results = trpc.useQueries((t) =>
    keyIds.map((keyId) =>
      t.api.keys.usageTimeseries(
        {
          startTime: anchor - HISTORICAL_DATA_WINDOW * 3,
          endTime: anchor,
          keyAuthId,
          keyId,
        },
        {
          staleTime: Number.POSITIVE_INFINITY,
          refetchOnWindowFocus: false,
        },
      ),
    ),
  );

  // `results` is a fresh array every render, so key the memo on what actually
  // changed. Rebuilding the Map on unrelated renders would hand every cell a
  // new context value and re-render the whole table for nothing.
  const signature = results.map((r) => `${r.status}:${r.dataUpdatedAt}`).join("|");

  // biome-ignore lint/correctness/useExhaustiveDependencies: `signature` stands in for `results`
  const byKeyId = useMemo(() => {
    const map = new Map<string, KeyTimeseries>();
    keyIds.forEach((keyId, index) => {
      const result = results[index];
      map.set(keyId, {
        timeseries: result?.data?.timeseries ? process(result.data.timeseries) : EMPTY_TIMESERIES,
        isLoading: result?.isLoading ?? false,
        isError: result?.isError ?? false,
      });
    });
    return map;
  }, [keyIds, signature]);

  return <KeyTimeseriesContext.Provider value={byKeyId}>{children}</KeyTimeseriesContext.Provider>;
};

export const useKeyTimeseries = (keyId: string): KeyTimeseries => {
  const map = useContext(KeyTimeseriesContext);
  if (!map) {
    // Falling back to empty-but-not-loading would render a green "Operational"
    // badge for every key without a request ever being made.
    throw new Error("useKeyTimeseries must be used within a KeyTimeseriesProvider");
  }
  // A key can legitimately be absent for a render or two while the row list and
  // `keyIds` converge; treat that as still loading rather than as no usage.
  return map.get(keyId) ?? PENDING;
};
