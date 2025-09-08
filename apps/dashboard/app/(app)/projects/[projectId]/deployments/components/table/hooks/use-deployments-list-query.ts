import { trpc } from "@/lib/trpc/client";
import type { Deployment } from "@/lib/trpc/routers/deploy/project/deployment/list";
import { useEffect, useMemo, useState } from "react";
import {
  type DeploymentListQuerySearchParams,
  deploymentListFilterFieldConfig,
  deploymentListFilterFieldNames,
} from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";

export function useDeploymentsListQuery() {
  const [totalCount, setTotalCount] = useState(0);
  const [deploymentsMap, setDeploymentsMap] = useState(() => new Map<string, Deployment>());
  const { filters } = useFilters();

  const deployments = useMemo(() => Array.from(deploymentsMap.values()), [deploymentsMap]);

  const queryParams = useMemo(() => {
    const params: DeploymentListQuerySearchParams = {
      status: [],
      environment: [],
      branch: [],
      startTime: null,
      endTime: null,
      since: null,
    };

    filters.forEach((filter) => {
      switch (filter.field) {
        case "status":
        case "environment": {
          if (!deploymentListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
            return;
          }
          const fieldConfig = deploymentListFilterFieldConfig[filter.field];
          const validOperators = fieldConfig.operators;
          if (!validOperators.includes(filter.operator)) {
            throw new Error("Invalid operator");
          }
          if (typeof filter.value === "string") {
            params[filter.field]?.push({
              operator: "is",
              value: filter.value,
            });
          }
          break;
        }
        case "branch": {
          if (!deploymentListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
            return;
          }
          const fieldConfig = deploymentListFilterFieldConfig[filter.field];
          const validOperators = fieldConfig.operators;
          if (!validOperators.includes(filter.operator)) {
            throw new Error("Invalid operator");
          }
          if (typeof filter.value === "string") {
            params[filter.field]?.push({
              operator: "contains",
              value: filter.value,
            });
          }
          break;
        }
        case "startTime":
        case "endTime":
          params[filter.field] = filter.value as number;
          break;
        case "since":
          params.since = filter.value as string;
          break;
      }
    });
    if (params.status?.length === 0) {
      params.status = null;
    }
    if (params.environment?.length === 0) {
      params.environment = null;
    }
    if (params.branch?.length === 0) {
      params.branch = null;
    }

    return params;
  }, [filters]);

  const {
    data: deploymentData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.deploy.project.deployment.list.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: 30_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (deploymentData) {
      const newMap = new Map<string, Deployment>();
      deploymentData.pages.forEach((page) => {
        page.deployments.forEach((deployment) => {
          newMap.set(deployment.id, deployment);
        });
      });
      if (deploymentData.pages.length > 0) {
        setTotalCount(deploymentData.pages[0].total);
      }
      setDeploymentsMap(newMap);
    }
  }, [deploymentData]);

  return {
    deployments,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
