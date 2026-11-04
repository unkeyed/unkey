import {
  type Deployment,
  type DeploymentListFilter,
  type Environment,
  useDeploymentsPage,
  useEnvironments,
} from "@/lib/deploy-data";

// Legacy pages only. Everything else reads through @/lib/deploy-data.
export function useDeployments(filter: DeploymentListFilter) {
  const environments = useEnvironments();
  const page = useDeploymentsPage(filter);
  const environmentById = new Map((environments.data ?? []).map((e) => [e.id, e]));
  const deployments = page.data?.pages.flatMap((p) => p.rows) ?? [];
  return {
    rows: deployments.map(
      (deployment): { deployment: Deployment; environment: Environment | undefined } => ({
        deployment,
        environment: environmentById.get(deployment.environmentId),
      }),
    ),
    isLoading: environments.isLoading || page.isLoading,
    isError: page.isError,
    hasNextPage: page.hasNextPage ?? false,
    isFetchingNextPage: page.isFetchingNextPage,
    fetchNextPage: page.fetchNextPage,
  };
}
