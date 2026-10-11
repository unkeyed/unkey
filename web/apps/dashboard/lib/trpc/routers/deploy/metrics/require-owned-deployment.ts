import { db } from "@/lib/db";
import {
  type DeploymentOwnershipCache,
  type OwnedDeployment,
  loadOwnedDeployment,
} from "@/lib/trpc/deployment-ownership";
import { TRPCError } from "@trpc/server";

function queryOwnedDeployment(
  workspaceId: string,
  deploymentId: string,
): Promise<OwnedDeployment | undefined> {
  return db.query.deployments.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.id, deploymentId), eq(table.workspaceId, workspaceId)),
    columns: {
      id: true,
      projectId: true,
      environmentId: true,
    },
  });
}

export async function requireOwnedDeployment(
  cache: DeploymentOwnershipCache,
  workspaceId: string,
  deploymentId: string,
  missingMessage: string,
): Promise<OwnedDeployment> {
  const deployment = await loadOwnedDeployment(
    cache,
    workspaceId,
    deploymentId,
    queryOwnedDeployment,
  );
  if (!deployment) {
    throw new TRPCError({ code: "NOT_FOUND", message: missingMessage });
  }
  return deployment;
}
