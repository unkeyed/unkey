import type { Deployment } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { useLiveInfiniteQuery } from "@tanstack/react-db";
import { useAppId, useProjectData } from "../../data-provider";
import { DEPLOYMENTS_PAGE_SIZE, deploymentsTableQueryFor } from "../../data-provider-queries";
import type { DeploymentListFilter } from "./deployment-list-input";

export type DeploymentListRow = {
  deployment: Deployment;
  environment: Environment | undefined;
};

export function useDeployments(filter: DeploymentListFilter) {
  const { projectId, environments, isEnvironmentsLoading } = useProjectData();
  const appId = useAppId();

  const query = useLiveInfiniteQuery(
    deploymentsTableQueryFor(projectId, appId, filter),
    {
      pageSize: DEPLOYMENTS_PAGE_SIZE,
      getNextPageParam: (lastPage, allPages) =>
        lastPage.length === DEPLOYMENTS_PAGE_SIZE ? allPages.length : undefined,
    },
    [projectId, appId, JSON.stringify(filter)],
  );

  const environmentById = new Map(environments.map((e) => [e.id, e]));
  const rows = query.data.map(
    (deployment): DeploymentListRow => ({
      deployment,
      environment: environmentById.get(deployment.environmentId),
    }),
  );

  return {
    rows,
    isLoading: isEnvironmentsLoading || query.isLoading,
    isError: query.isError,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
  };
}
