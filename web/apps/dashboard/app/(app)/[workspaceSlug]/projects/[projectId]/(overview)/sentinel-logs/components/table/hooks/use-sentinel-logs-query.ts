"use client";

import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { useProject } from "../../../../layout-provider";
import { useSentinelLogsFilters } from "../../../hooks/use-sentinel-logs-filters";

type UseSentinelLogsQueryParams = {
  limit?: number;
};

export function useSentinelLogsQuery({ limit = 50 }: UseSentinelLogsQueryParams = {}) {
  const { projectId } = useProject();
  const { filters } = useSentinelLogsFilters();

  const queryInput = useMemo(() => {
    // Extract filters
    const statusFilters = filters.filter((f) => f.field === "status").map((f) => Number(f.value));

    const methodFilters = filters.filter((f) => f.field === "methods").map((f) => String(f.value));

    const pathFilters = filters
      .filter((f) => f.field === "paths")
      .map((f) => ({
        operator: "contains" as const,
        value: String(f.value),
      }));

    const deploymentIdFilter = filters.find((f) => f.field === "deploymentId");
    const environmentIdFilter = filters.find((f) => f.field === "environmentId");

    // Extract time filters
    const startTimeFilter = filters.find((f) => f.field === "startTime");
    const endTimeFilter = filters.find((f) => f.field === "endTime");
    const sinceFilter = filters.find((f) => f.field === "since");

    return {
      projectId,
      deploymentId: deploymentIdFilter ? String(deploymentIdFilter.value) : null,
      environmentId: environmentIdFilter ? String(environmentIdFilter.value) : null,
      limit,
      startTime: startTimeFilter ? Number(startTimeFilter.value) : Date.now() - 6 * 60 * 60 * 1000,
      endTime: endTimeFilter ? Number(endTimeFilter.value) : Date.now(),
      since: sinceFilter ? String(sinceFilter.value) : "6h",
      statusCodes: statusFilters.length > 0 ? statusFilters : null,
      methods: methodFilters.length > 0 ? methodFilters : null,
      paths: pathFilters.length > 0 ? pathFilters : null,
    };
  }, [filters, limit, projectId]);

  const { data, isLoading, error, hasNextPage, fetchNextPage, isFetchingNextPage } =
    trpc.deploy.sentinelLogs.query.useInfiniteQuery(queryInput, {
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
