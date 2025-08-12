import { trpc } from "@/lib/trpc/client";
import type { Deployment } from "@/lib/trpc/routers/deploy/project/deployment/list";
import { useEffect, useMemo, useState } from "react";

export function useDeploymentsListQuery() {
  const [totalCount, setTotalCount] = useState(0);
  const [deploymentsMap, setDeploymentsMap] = useState(
    () => new Map<string, Deployment>()
  );

  const deployments = useMemo(
    () => Array.from(deploymentsMap.values()),
    [deploymentsMap]
  );

  const {
    data: deploymentData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.deploy.deployment.list.useInfiniteQuery(
    {}, // No query params since we removed filters
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: 30000, // 30 seconds
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }
  );

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
