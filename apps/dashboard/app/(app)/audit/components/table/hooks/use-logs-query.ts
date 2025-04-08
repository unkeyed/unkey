import { trpc } from "@/lib/trpc/client";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { useEffect, useMemo, useState } from "react";
import { useFilters } from "../../../hooks/use-filters";
import { type AuditQueryLogsPayload, DEFAULT_BUCKET_NAME } from "../query-logs.schema";

type UseLogsQueryParams = {
  limit?: number;
};

export function useAuditLogsQuery({ limit = 50 }: UseLogsQueryParams) {
  const [historicalLogsMap, setHistoricalLogsMap] = useState(() => new Map<string, AuditLog>());

  const { filters } = useFilters();

  const historicalLogs = useMemo(() => Array.from(historicalLogsMap.values()), [historicalLogsMap]);

  const queryParams = useMemo(() => {
    const params: AuditQueryLogsPayload = {
      limit,
      startTime: undefined,
      endTime: undefined,
      events: { filters: [] },
      users: { filters: [] },
      rootKeys: { filters: [] },
      since: "",
      bucket: DEFAULT_BUCKET_NAME,
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "events": {
          if (typeof filter.value !== "string") {
            console.error("Events filter value type has to be 'string'");
            return;
          }
          params.events?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }
        case "rootKeys": {
          if (typeof filter.value !== "string") {
            console.error("RootKeys filter value type has to be 'string'");
            return;
          }
          params.rootKeys?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "users": {
          if (typeof filter.value !== "string") {
            console.error("Users filter value type has to be 'string'");
            return;
          }
          params.users?.filters.push({
            operator: filter.operator,
            value: filter.value,
          });
          break;
        }

        case "startTime":
        case "endTime": {
          if (typeof filter.value !== "number") {
            console.error(`${filter.field} filter value type has to be 'string'`);
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

        case "bucket": {
          if (typeof filter.value !== "string") {
            console.error("Bucket filter value type has to be 'string'");
            return;
          }
          params.bucket = filter.value;
          break;
        }
      }
    });

    return params;
  }, [filters, limit]);

  const {
    data: initialData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = trpc.audit.logs.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Update historical logs effect
  useEffect(() => {
    if (initialData) {
      const newMap = new Map<string, AuditLog>();
      initialData.pages.forEach((page) => {
        page.auditLogs.forEach((log) => {
          newMap.set(log.auditLog.id, log);
        });
      });
      setHistoricalLogsMap(newMap);
    }
  }, [initialData]);

  return {
    historicalLogs,
    isLoading,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
