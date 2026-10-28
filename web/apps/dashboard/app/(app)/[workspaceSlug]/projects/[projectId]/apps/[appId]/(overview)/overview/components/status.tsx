import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatusGroup,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { summarizeInstances } from "@/lib/collections/deploy/instance-status";

export type StatusBadgeVariant = "primary" | "success" | "error" | "warning";

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

export const GROUP_BADGE_VARIANT: Record<DeploymentStatusGroup, StatusBadgeVariant> = {
  ready: "primary",
  building: "primary",
  queued: "primary",
  blocked: "warning",
  failed: "error",
  stopped: "primary",
  cancelled: "primary",
  superseded: "primary",
  skipped: "primary",
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
  { label: string; dotClass: string; badge: StatusBadgeVariant; polls: boolean }
> = {
  live: {
    label: "Live",
    dotClass: DEPLOYMENT_GROUP_COLOR.ready,
    badge: "success",
    polls: true,
  },
  deploying: {
    label: DEPLOYMENT_STATUS_LABELS.deploying,
    dotClass: DEPLOYMENT_GROUP_COLOR.building,
    badge: "primary",
    polls: true,
  },
  crashing: {
    label: "Crashing",
    dotClass: DEPLOYMENT_GROUP_COLOR.failed,
    badge: "error",
    polls: true,
  },
  failed: {
    label: DEPLOYMENT_STATUS_LABELS.failed,
    dotClass: DEPLOYMENT_GROUP_COLOR.failed,
    badge: "error",
    polls: false,
  },
  stopped: {
    label: DEPLOYMENT_STATUS_LABELS.stopped,
    dotClass: DEPLOYMENT_GROUP_COLOR.stopped,
    badge: "primary",
    polls: false,
  },
  cancelled: {
    label: DEPLOYMENT_STATUS_LABELS.cancelled,
    dotClass: DEPLOYMENT_GROUP_COLOR.cancelled,
    badge: "primary",
    polls: false,
  },
  superseded: {
    label: DEPLOYMENT_STATUS_LABELS.superseded,
    dotClass: DEPLOYMENT_GROUP_COLOR.superseded,
    badge: "primary",
    polls: false,
  },
  skipped: {
    label: DEPLOYMENT_STATUS_LABELS.skipped,
    dotClass: DEPLOYMENT_GROUP_COLOR.skipped,
    badge: "primary",
    polls: false,
  },
};
