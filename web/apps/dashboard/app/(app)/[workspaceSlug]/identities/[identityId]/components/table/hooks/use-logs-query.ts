import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { useLiveTail } from "@/hooks/use-live-tail";
import { shownTotalOf, tableStateOf, useInfiniteList } from "@/lib/loadable/list";
import { trpc, trpcClient } from "@/lib/trpc/client";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { useQueryTime } from "@/providers/query-time-provider";
import { getQueryKey } from "@trpc/react-query";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { identityDetailsFilterFieldConfig } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import type { IdentityLogsPayload } from "../query-logs.schema";

const logsOf = (page: { logs: IdentityLog[] }) => page.logs;
const requestIdOf = (log: IdentityLog) => log.request_id;

type UseIdentityLogsQueryProps = {
  identityId: string;
  startPolling?: boolean;
  pollIntervalMs?: number;
};

export const useIdentityLogsQuery = ({
  identityId,
  startPolling = false,
  pollIntervalMs = 2000,
}: UseIdentityLogsQueryProps) => {
  const { filters } = useFilters();
  const { queryTime: timestamp } = useQueryTime();

  const queryParams = useMemo(() => {
    const params: IdentityLogsPayload = {
      identityId,
      limit: 50,
      startTime: timestamp - HISTORICAL_DATA_WINDOW,
      endTime: timestamp,
      since: "",
      cursor: null,
      tags: null,
      outcomes: null,
    };

    filters.forEach((filter) => {
      if (!(filter.field in identityDetailsFilterFieldConfig)) {
        return;
      }

      switch (filter.field) {
        case "tags": {
          if (typeof filter.value === "string" && filter.value.trim()) {
            const fieldConfig = identityDetailsFilterFieldConfig[filter.field];
            const validOperators = fieldConfig.operators;

            const operator = validOperators.includes(filter.operator)
              ? filter.operator
              : validOperators[0];

            params.tags = [
              {
                operator,
                value: filter.value,
              },
            ];
          }
          break;
        }

        case "startTime":
        case "endTime": {
          // TypeScript knows filter.value is number for these fields
          params[filter.field] = filter.value;
          break;
        }

        case "since": {
          // TypeScript knows filter.value is string for this field
          params.since = filter.value;
          break;
        }

        case "outcomes": {
          type ValidOutcome = (typeof KEY_VERIFICATION_OUTCOMES)[number];
          if (
            typeof filter.value === "string" &&
            KEY_VERIFICATION_OUTCOMES.includes(filter.value as ValidOutcome)
          ) {
            if (!params.outcomes) {
              params.outcomes = [];
            }
            params.outcomes.push({
              operator: "is",
              value: filter.value as ValidOutcome,
            });
          }
          break;
        }
      }
    });

    return params;
  }, [filters, timestamp, identityId]);

  // Main query for historical data
  const query = trpc.identity.logs.query.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    refetchInterval: false,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    trpc: {
      context: {
        skipBatch: true,
      },
    },
  });

  const table = tableStateOf(useInfiniteList(query, logsOf, { idOf: requestIdOf }));
  const historicalLogs = table.rows;
  const realtimeLogs = useLiveTail({
    enabled: startPolling,
    intervalMs: pollIntervalMs,
    queryKey: getQueryKey(trpc.identity.logs.query, queryParams),
    fetchNewest: async (signal) => {
      const page = await trpcClient.identity.logs.query.query(
        { ...queryParams, endTime: Date.now() },
        { signal },
      );
      return page.logs;
    },
    keyOf: requestIdOf,
    historical: historicalLogs,
  });

  return {
    realtimeLogs,
    table,
    totalCount: shownTotalOf({
      loaded: historicalLogs.length,
      total: null,
      hasMore: query.hasNextPage === true,
    }),
  };
};
