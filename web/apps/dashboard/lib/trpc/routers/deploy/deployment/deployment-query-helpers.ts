import type { InstanceStatus } from "@/lib/collections/deploy/instance-status";
import type { InferSelectModel } from "@/lib/db";
import type { LastExit } from "@/lib/types/deploy";
import { type ContainerStatus, deployments } from "@unkey/db/src/schema";
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

export const deploymentListSelect = {
  ...deploymentSelectFields,
  appId: deployments.appId,
} as const;

export type DeploymentListSelection = Pick<
  InferSelectModel<typeof deployments>,
  keyof typeof deploymentListSelect
>;

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

// computeLastExit picks the most recent exit across all instances of a
// deployment so that a multi-region rollout with one OOM-ing pod still
// surfaces the failure even when others are healthy. Tie-break by
// finishedAt (preferred) and fall back to the live waiting reason
// (CrashLoopBackOff, ImagePullBackOff, …) when there's no exit yet.
// Returns null when no instance has reported a termination or waiting
// reason (healthy deployments). Shared by the list and getById routes so
// both surface the same header badge data.
export function computeLastExit(
  rows: { containerStatus: ContainerStatus | null }[],
): LastExit | null {
  let result: LastExit | null = null;
  for (const row of rows) {
    const status = row.containerStatus ?? ({} as ContainerStatus);
    const term = status.lastTerminationState ?? null;
    const waiting = status.waiting ?? null;
    const candidate: LastExit = {
      restartCount: status.restartCount ?? 0,
      exitCode: term?.exitCode ?? null,
      signal: term?.signal ?? null,
      reason: term?.reason ?? null,
      finishedAt: term?.finishedAt ?? null,
      statusReason: waiting?.reason ?? null,
    };
    if (candidate.reason === null && candidate.statusReason === null) {
      continue;
    }
    if (!result) {
      result = candidate;
      continue;
    }
    // Prefer the candidate with a more recent finishedAt; if neither has
    // one (only statusReason populated) keep whichever has higher
    // restartCount as a coarse recency tiebreaker.
    const prevTs = result.finishedAt ?? -1;
    const candTs = candidate.finishedAt ?? -1;
    if (candTs > prevTs || (candTs === prevTs && candidate.restartCount > result.restartCount)) {
      result = candidate;
    }
  }
  return result;
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
