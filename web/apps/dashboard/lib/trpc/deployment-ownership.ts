export type OwnedDeployment = {
  id: string;
  projectId: string;
  environmentId: string;
};

export type DeploymentOwnershipCache = Map<
  string,
  Map<string, Promise<OwnedDeployment | undefined>>
>;

type FindOwnedDeployment = (
  workspaceId: string,
  deploymentId: string,
) => Promise<OwnedDeployment | undefined>;

export function createDeploymentOwnershipCache(): DeploymentOwnershipCache {
  return new Map();
}

// Batched tRPC procedures share the cache created for the HTTP request. Each
// deployment metrics procedure checks ownership before reading ClickHouse, so
// remembering the in-flight lookup keeps that check to one deployments query.
export function loadOwnedDeployment(
  cache: DeploymentOwnershipCache,
  workspaceId: string,
  deploymentId: string,
  find: FindOwnedDeployment,
): Promise<OwnedDeployment | undefined> {
  let byDeployment = cache.get(workspaceId);
  if (!byDeployment) {
    byDeployment = new Map();
    cache.set(workspaceId, byDeployment);
  }

  const existing = byDeployment.get(deploymentId);
  if (existing) {
    return existing;
  }

  const pending = find(workspaceId, deploymentId);
  byDeployment.set(deploymentId, pending);
  return pending;
}
