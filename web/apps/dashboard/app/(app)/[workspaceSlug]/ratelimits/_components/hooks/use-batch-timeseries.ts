import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { useMemo } from "react";

export type NamespaceTimeseries = Array<{
  displayX: string;
  originalTimestamp: number;
  success: number;
  error: number;
  total: number;
}>;

export const useBatchRatelimitTimeseries = (namespaceIds: string[]) => {
  const { queryTime: timestamp } = useQueryTime();

  const startTime = timestamp - HISTORICAL_DATA_WINDOW;
  const endTime = timestamp;

  // The namespace list page is always a live view (no user-controlled time range),
  // so we auto-refresh every 10s to keep charts current.
  const { data, isLoading, isError } =
    trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery(
      { namespaceIds, startTime, endTime },
      {
        enabled: namespaceIds.length > 0,
        refetchInterval: 10_000,
      },
    );

  const timeseriesByNamespace = useMemo(() => {
    if (!data) {
      return {};
    }

    const result: Record<string, NamespaceTimeseries> = {};
    for (const [nsId, points] of Object.entries(data.timeseriesByNamespace)) {
      result[nsId] = points.map((ts) => ({
        displayX: formatTimestampForChart(ts.x, data.granularity),
        originalTimestamp: ts.x,
        success: ts.y.passed,
        error: ts.y.total - ts.y.passed,
        total: ts.y.total,
      }));
    }
    return result;
  }, [data]);

  return { timeseriesByNamespace, isLoading, isError, granularity: data?.granularity };
};
