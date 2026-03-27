import { keysOverviewFilterFieldConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useSort } from "@/components/logs/hooks/use-sort";
import { trpc } from "@/lib/trpc/client";
import { useQueryTime } from "@/providers/query-time-provider";
import { KEY_VERIFICATION_OUTCOMES, type KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import type { KeysQueryOverviewLogsPayload, SortFields } from "../schema/keys-overview.schema";

type UseLogsQueryParams = {
  limit?: number;
  apiId: string;
};

export function useKeysOverviewLogsQuery({ apiId, limit = 50 }: UseLogsQueryParams) {
  const { filters } = useFilters();
  const { sorts } = useSort<SortFields>();

  const { queryTime: timestamp } = useQueryTime();

  // Check if user explicitly set a time frame filter
  const hasTimeFrameFilter = useMemo(() => {
    return filters.some((filter) => filter.field === "startTime" || filter.field === "endTime");
  }, [filters]);

  const queryParams = useMemo(() => {
    const params: KeysQueryOverviewLogsPayload = {
      limit,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      keyIds: [],
      outcomes: [],
      identities: [],
      names: [],
      tags: [],
      apiId,
      since: "",
      sorts: sorts.length > 0 ? sorts : null,
      // Flag to indicate if user explicitly filtered by time frame
      // If true, use new logic to find keys with ANY usage in the time frame
      // If false or undefined, use the MV directly for speed
      useTimeFrameFilter: hasTimeFrameFilter,
    };

    filters.forEach((filter) => {
      const fieldConfig = keysOverviewFilterFieldConfig[filter.field];
      const validOperators = fieldConfig.operators;

      const operator = validOperators.includes(filter.operator)
        ? filter.operator
        : validOperators[0];

      switch (filter.field) {
        case "keyIds": {
          if (typeof filter.value === "string") {
            const keyIdOperator = operator === "is" || operator === "contains" ? operator : "is";

            params.keyIds?.push({
              operator: keyIdOperator,
              value: filter.value,
            });
          }
          break;
        }

        case "names":
        case "identities":
          if (typeof filter.value === "string") {
            params[filter.field]?.push({
              operator,
              value: filter.value,
            });
          }
          break;

        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            params.outcomes?.push({
              operator: "is",
              value: filter.value as ValidOutcome,
            });
          }
          break;
        }

        case "tags": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            params.tags?.push({
              operator,
              value: filter.value,
            });
          }
          break;
        }

        case "startTime":
        case "endTime": {
          const numValue =
            typeof filter.value === "number"
              ? filter.value
              : typeof filter.value === "string"
                ? Number(filter.value)
                : Number.NaN;

          if (!Number.isNaN(numValue)) {
            params[filter.field] = numValue;
          }
          break;
        }

        case "since":
          if (typeof filter.value === "string") {
            params.since = filter.value;
          }
          break;
      }
    });

    return params;
  }, [filters, limit, timestamp, apiId, sorts, hasTimeFrameFilter]);

  // Main query for historical data
  const {
    data: initialData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.api.keys.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Use request_id as the unique key since key_id might not be unique across different requests
  const historicalLogs = useMemo(() => {
    if (!initialData) {
      return [];
    }
    const map = new Map<string, KeysOverviewLog>();
    initialData.pages.forEach((page) => {
      page.keysOverviewLogs.forEach((log) => {
        map.set(log.request_id, log);
      });
    });
    return Array.from(map.values());
  }, [initialData]);

  return {
    historicalLogs,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
