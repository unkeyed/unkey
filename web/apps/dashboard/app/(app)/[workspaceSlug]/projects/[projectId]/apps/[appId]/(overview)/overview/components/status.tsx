import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";

export type DeploymentDisplayStatus = "live" | "deploying" | "crashing" | "failed" | "stopped";

export function deriveProductionStatus(deployment: Deployment): DeploymentDisplayStatus {
  if (deployment.status === "stopped") {
    return "stopped";
  }
  if (deployment.status === "failed") {
    return "failed";
  }
  const crashing =
    deployment.lastExit?.statusReason === "CrashLoopBackOff" ||
    (deployment.instances ?? []).some((i) => i.status === "failed");
  if (crashing) {
    return "crashing";
  }
  if (isDeploymentInFlight(deployment.status) || deployment.status === "awaiting_approval") {
    return "deploying";
  }
  return "live";
}

export const STATUS_META: Record<DeploymentDisplayStatus, { label: string; dotClass: string }> = {
  live: { label: "Live", dotClass: DEPLOYMENT_GROUP_COLOR.ready },
  deploying: {
    label: DEPLOYMENT_STATUS_LABELS.deploying,
    dotClass: DEPLOYMENT_GROUP_COLOR.building,
  },
  crashing: { label: "Crashing", dotClass: DEPLOYMENT_GROUP_COLOR.failed },
  failed: { label: DEPLOYMENT_STATUS_LABELS.failed, dotClass: DEPLOYMENT_GROUP_COLOR.failed },
  stopped: { label: DEPLOYMENT_STATUS_LABELS.stopped, dotClass: DEPLOYMENT_GROUP_COLOR.stopped },
};
