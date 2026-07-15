import type { InstanceStatus } from "@/lib/collections/deploy/instance-status";
import { and, db, eq } from "@/lib/db";
import { apps, deployments } from "@unkey/db/src/schema";
import { mapRegionToFlag } from "../network/utils";

export const deploymentSelectFields = {
  id: deployments.id,
  projectId: deployments.projectId,
  environmentId: deployments.environmentId,
  gitCommitSha: deployments.gitCommitSha,
  gitBranch: deployments.gitBranch,
  gitCommitMessage: deployments.gitCommitMessage,
  gitCommitAuthorHandle: deployments.gitCommitAuthorHandle,
  gitCommitAuthorAvatarUrl: deployments.gitCommitAuthorAvatarUrl,
  gitCommitTimestamp: deployments.gitCommitTimestamp,
  prNumber: deployments.prNumber,
  forkRepositoryFullName: deployments.forkRepositoryFullName,
  image: deployments.image,
  status: deployments.status,
  desiredState: deployments.desiredState,
  trigger: deployments.trigger,
  triggeredBy: deployments.triggeredBy,
  triggerReason: deployments.triggerReason,
  cpuMillicores: deployments.cpuMillicores,
  memoryMib: deployments.memoryMib,
  storageMib: deployments.storageMib,
  port: deployments.port,
  upstreamProtocol: deployments.upstreamProtocol,
  healthcheck: deployments.healthcheck,
  shutdownSignal: deployments.shutdownSignal,
  createdAt: deployments.createdAt,
  updatedAt: deployments.updatedAt,
} as const;

export function mapInstanceRow(row: {
  id: string;
  regionId: string;
  regionName: string;
  regionPlatform: string;
  status: InstanceStatus;
}) {
  return {
    id: row.id,
    region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
    flagCode: mapRegionToFlag(row.regionName),
    status: row.status,
  };
}

export function normalizeDeploymentRow(deployment: {
  gitBranch: string | null;
  prNumber: number | null;
  forkRepositoryFullName: string | null;
  gitCommitAuthorAvatarUrl: string | null;
  gitCommitTimestamp: number | null;
}) {
  return {
    gitBranch: deployment.gitBranch ?? "",
    prNumber: deployment.prNumber ?? null,
    forkRepositoryFullName: deployment.forkRepositoryFullName ?? null,
    gitCommitAuthorAvatarUrl:
      deployment.gitCommitAuthorAvatarUrl ?? "https://github.com/identicons/dummy-user.png",
    gitCommitTimestamp: deployment.gitCommitTimestamp,
  };
}

// The overview resolves the live deployment by id from the collection, so it
// must be present even when older than the newest-N window the list returns.
export async function fetchCurrentDeploymentOutsideWindow(
  workspaceId: string,
  input: { projectId: string; appId: string },
  loadedRows: { id: string }[],
) {
  const [app] = await db
    .select({ currentDeploymentId: apps.currentDeploymentId })
    .from(apps)
    .where(
      and(
        eq(apps.workspaceId, workspaceId),
        eq(apps.projectId, input.projectId),
        eq(apps.id, input.appId),
      ),
    );
  const currentId = app?.currentDeploymentId;
  if (!currentId || loadedRows.some((d) => d.id === currentId)) {
    return null;
  }
  const [deployment] = await db
    .select({ ...deploymentSelectFields, appId: deployments.appId })
    .from(deployments)
    .where(
      and(
        eq(deployments.workspaceId, workspaceId),
        eq(deployments.projectId, input.projectId),
        eq(deployments.id, currentId),
      ),
    );
  return deployment ?? null;
}
