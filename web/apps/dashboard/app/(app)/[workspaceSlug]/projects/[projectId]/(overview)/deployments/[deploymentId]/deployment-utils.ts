import type { DeploymentStatus } from "@/lib/collections";
import type { StepsData } from "./(deployment-progress)/deployment-progress";

const DEPLOYMENT_STATUSES: ReadonlySet<string> = new Set<DeploymentStatus>([
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "ready",
  "failed",
]);

function isDeploymentStatus(value: string): value is DeploymentStatus {
  return DEPLOYMENT_STATUSES.has(value);
}

export function deriveStatusFromSteps(
  steps: StepsData | undefined,
  fallback: string,
): DeploymentStatus {
  if (!steps) {
    return isDeploymentStatus(fallback) ? fallback : "pending";
  }

  const { queued, building, deploying, network, finalizing, starting } = steps;

  if ([queued, building, deploying, network, finalizing, starting].some((s) => s?.error)) {
    return "failed";
  }
  if (finalizing && !finalizing.endedAt) {
    return "finalizing";
  }
  if (finalizing?.completed) {
    return "ready";
  }
  if (network && !network.endedAt) {
    return "network";
  }
  if (deploying && !deploying.endedAt) {
    return "deploying";
  }
  if (building && !building.endedAt) {
    return "building";
  }
  if (starting && !starting.endedAt) {
    return "starting";
  }
  if (queued && !queued.endedAt) {
    return "pending";
  }

  return isDeploymentStatus(fallback) ? fallback : "pending";
}
