"use client";

import { trpc } from "@/lib/trpc/client";
import { useParams } from "next/navigation";
import { useMemo } from "react";
import { DEFAULT_LIMIT } from "../../../constants";
import type { RuntimeLogsFilter } from "../../../types";

type UseRuntimeLogsQueryParams = {
  limit?: number;
  filters: RuntimeLogsFilter[];
};

export function useRuntimeLogsQuery({ limit = DEFAULT_LIMIT, filters }: UseRuntimeLogsQueryParams) {
  const params = useParams<{ projectId: string; deploymentId: string }>();

  // Transform filters to tRPC input format
  const queryInput = useMemo(() => {
    const severityFilters = filters
      .filter((f) => f.field === "severity")
      .map((f) => ({ operator: "is" as const, value: String(f.value) }));

    const messageFilter = filters.find((f) => f.field === "message");
    const startTimeFilter = filters.find((f) => f.field === "startTime");
    const endTimeFilter = filters.find((f) => f.field === "endTime");
    const sinceFilter = filters.find((f) => f.field === "since");

    return {
      projectId: params.projectId,
      deploymentId: params.deploymentId,
      limit,
      startTime: startTimeFilter ? Number(startTimeFilter.value) : Date.now() - 6 * 60 * 60 * 1000,
      endTime: endTimeFilter ? Number(endTimeFilter.value) : Date.now(),
      since: sinceFilter ? String(sinceFilter.value) : "6h",
      severity: severityFilters.length > 0 ? { filters: severityFilters } : null,
      message: messageFilter ? String(messageFilter.value) : null,
    };
  }, [filters, limit, params]);

  const { data, isLoading, error, hasNextPage, fetchNextPage, isFetchingNextPage } =
    trpc.deploy.runtimeLogs.query.useInfiniteQuery(queryInput, {
      getNextPageParam: (lastPage) => (lastPage.hasMore ? lastPage.nextCursor : undefined),
    });

  const logs = useMemo(() => {
    return data?.pages.flatMap((page) => page.logs) ?? [];
  }, [data]);

  const total = data?.pages[0]?.total ?? 0;

  return {
    logs,
    total,
    isLoading,
    error,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
  };
}
