import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { summarizeInstances } from "@/lib/collections/deploy/instance-status";
import type { NodeTone } from "./canvas/nodes";

export type DeploymentDisplayStatus =
  | "live"
  | "deploying"
  | "crashing"
  | "failed"
  | "stopped"
  | "cancelled"
  | "superseded"
  | "skipped";

export function deriveProductionStatus(
  deployment: Pick<Deployment, "status" | "instances" | "targetRegions" | "desiredState">,
): DeploymentDisplayStatus {
  if (
    deployment.status === "stopped" ||
    deployment.status === "failed" ||
    deployment.status === "cancelled" ||
    deployment.status === "superseded" ||
    deployment.status === "skipped"
  ) {
    return deployment.status;
  }
  if (summarizeInstances(deployment)?.state === "unhealthy") {
    return "crashing";
  }
  if (isDeploymentInFlight(deployment.status) || deployment.status === "awaiting_approval") {
    return "deploying";
  }
  return "live";
}

export const STATUS_META: Record<
  DeploymentDisplayStatus,
  { label: string; dotClass: string; tone: NodeTone }
> = {
  live: { label: "Live", dotClass: DEPLOYMENT_GROUP_COLOR.ready, tone: "default" },
  deploying: {
    label: DEPLOYMENT_STATUS_LABELS.deploying,
    dotClass: DEPLOYMENT_GROUP_COLOR.building,
    tone: "default",
  },
  crashing: { label: "Crashing", dotClass: DEPLOYMENT_GROUP_COLOR.failed, tone: "error" },
  failed: {
    label: DEPLOYMENT_STATUS_LABELS.failed,
    dotClass: DEPLOYMENT_GROUP_COLOR.failed,
    tone: "error",
  },
  stopped: {
    label: DEPLOYMENT_STATUS_LABELS.stopped,
    dotClass: DEPLOYMENT_GROUP_COLOR.stopped,
    tone: "default",
  },
  cancelled: {
    label: DEPLOYMENT_STATUS_LABELS.cancelled,
    dotClass: DEPLOYMENT_GROUP_COLOR.cancelled,
    tone: "default",
  },
  superseded: {
    label: DEPLOYMENT_STATUS_LABELS.superseded,
    dotClass: DEPLOYMENT_GROUP_COLOR.superseded,
    tone: "default",
  },
  skipped: {
    label: DEPLOYMENT_STATUS_LABELS.skipped,
    dotClass: DEPLOYMENT_GROUP_COLOR.skipped,
    tone: "default",
  },
};
