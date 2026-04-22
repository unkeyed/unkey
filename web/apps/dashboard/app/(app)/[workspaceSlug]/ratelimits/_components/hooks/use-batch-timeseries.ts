import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/hooks/use-filters";
import { formatTimestampForChart } from "@/components/logs/chart/utils/format-timestamp";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { trpc } from "@/lib/trpc/client";
import { getTimestampFromRelative } from "@/lib/utils";
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
  const { filters } = useFilters();

  const { startTime, endTime } = useMemo(() => {
    let start = timestamp - HISTORICAL_DATA_WINDOW;
    let end = timestamp;

    for (const filter of filters) {
      switch (filter.field) {
        case "since": {
          if (typeof filter.value === "string" && filter.value !== "") {
            start = getTimestampFromRelative(filter.value);
            end = timestamp;
          }
          break;
        }
        case "startTime": {
          if (typeof filter.value === "number") {
            start = filter.value;
          }
          break;
        }
        case "endTime": {
          if (typeof filter.value === "number") {
            end = filter.value;
          }
          break;
        }
      }
    }

    return { startTime: start, endTime: end };
  }, [filters, timestamp]);

  const { data, isLoading, isError } = trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery(
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
    // Seed every requested namespace with an empty array so the chart can
    // distinguish "query completed but no data" from "still loading".
    for (const nsId of namespaceIds) {
      result[nsId] = [];
    }
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
  }, [data, namespaceIds]);

  return { timeseriesByNamespace, isLoading, isError, granularity: data?.granularity };
};
