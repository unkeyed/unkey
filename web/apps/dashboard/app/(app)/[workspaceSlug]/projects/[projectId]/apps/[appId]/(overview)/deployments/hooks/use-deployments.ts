import type { Deployment } from "@/lib/collections";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import type { Environment } from "@/lib/collections/deploy/environments";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { buildDeploymentListInput } from "./deployment-list-input";
import { useFilters } from "./use-filters";

export const DEPLOYMENTS_PAGE_SIZE = 25;

export type DeploymentListRow = {
  deployment: Deployment;
  environment: Environment | undefined;
};

export function useDeployments(pageSize: number = DEPLOYMENTS_PAGE_SIZE) {
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();
  const { filters } = useFilters();

  const { input, cannotMatch } = useMemo(
    () => buildDeploymentListInput(filters, environments),
    [filters, environments],
  );

  const query = trpc.deploy.deployment.list.useInfiniteQuery(
    { projectId, appId, ...input, limit: pageSize },
    {
      enabled: !isEnvironmentsLoading && !cannotMatch,
      // Filters are part of the query input, so a change starts a new query;
      // the previous rows stay on screen instead of a skeleton flash.
      keepPreviousData: true,
      getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
      // Polling is opt-in while a build runs; idle pages rely on focus refetch
      // and post-mutation invalidation. A refetch reloads every opened page,
      // so only the newest page decides.
      refetchInterval: (data) =>
        data?.pages[0]?.deployments.some((d) => isDeploymentInFlight(d.status)) ? 5_000 : false,
    },
  );

  const rows = useMemo((): DeploymentListRow[] => {
    const environmentById = new Map(environments.map((e) => [e.id, e]));
    return (query.data?.pages ?? []).flatMap((page) =>
      page.deployments.map((deployment) => ({
        deployment,
        environment: environmentById.get(deployment.environmentId),
      })),
    );
  }, [query.data, environments]);

  return {
    rows,
    isLoading: isEnvironmentsLoading || query.isInitialLoading,
    isError: query.isError,
    refetch: query.refetch,
    isFiltered: filters.length > 0,
    hasNextPage: query.hasNextPage ?? false,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
  };
}
