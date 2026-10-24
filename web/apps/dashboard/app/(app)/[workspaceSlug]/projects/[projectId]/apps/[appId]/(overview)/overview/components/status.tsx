import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatusGroup,
  statusGroupOf,
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

const DISPLAY_BY_GROUP: Record<DeploymentStatusGroup, DeploymentDisplayStatus> = {
  ready: "live",
  building: "deploying",
  queued: "deploying",
  blocked: "deploying",
  failed: "failed",
  stopped: "stopped",
  cancelled: "cancelled",
  superseded: "superseded",
  skipped: "skipped",
};

export function deriveProductionStatus(
  deployment: Pick<Deployment, "status" | "instances" | "targetRegions" | "desiredState">,
): DeploymentDisplayStatus {
  const display = DISPLAY_BY_GROUP[statusGroupOf(deployment.status)];
  if (
    (display === "live" || display === "deploying") &&
    summarizeInstances(deployment)?.state === "unhealthy"
  ) {
    return "crashing";
  }
  return display;
}

export const STATUS_META: Record<
  DeploymentDisplayStatus,
  { label: string; dotClass: string; tone: NodeTone; polls: boolean }
> = {
  live: { label: "Live", dotClass: DEPLOYMENT_GROUP_COLOR.ready, tone: "default", polls: true },
  deploying: {
    label: DEPLOYMENT_STATUS_LABELS.deploying,
    dotClass: DEPLOYMENT_GROUP_COLOR.building,
    tone: "default",
    polls: true,
  },
  crashing: {
    label: "Crashing",
    dotClass: DEPLOYMENT_GROUP_COLOR.failed,
    tone: "error",
    polls: true,
  },
  failed: {
    label: DEPLOYMENT_STATUS_LABELS.failed,
    dotClass: DEPLOYMENT_GROUP_COLOR.failed,
    tone: "error",
    polls: false,
  },
  stopped: {
    label: DEPLOYMENT_STATUS_LABELS.stopped,
    dotClass: DEPLOYMENT_GROUP_COLOR.stopped,
    tone: "default",
    polls: false,
  },
  cancelled: {
    label: DEPLOYMENT_STATUS_LABELS.cancelled,
    dotClass: DEPLOYMENT_GROUP_COLOR.cancelled,
    tone: "default",
    polls: false,
  },
  superseded: {
    label: DEPLOYMENT_STATUS_LABELS.superseded,
    dotClass: DEPLOYMENT_GROUP_COLOR.superseded,
    tone: "default",
    polls: false,
  },
  skipped: {
    label: DEPLOYMENT_STATUS_LABELS.skipped,
    dotClass: DEPLOYMENT_GROUP_COLOR.skipped,
    tone: "default",
    polls: false,
  },
};
