import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
  deploymentStatusColor,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import { type InstanceSummary, instanceLabel } from "@/lib/collections/deploy/instance-status";

export function runningCountText(summary: InstanceSummary): string {
  return `${summary.running} of ${summary.total} running`;
}

export type PageStatus = {
  label: string;
  dotClass: string;
  inFlight: boolean;
};

export function resolvePageStatus(
  status: DeploymentStatus,
  instances: InstanceSummary | null,
): PageStatus {
  if (status === "ready" && instances?.state === "unhealthy") {
    return {
      label: instanceLabel.unhealthy,
      dotClass: DEPLOYMENT_GROUP_COLOR.failed,
      inFlight: false,
    };
  }
  return {
    label: DEPLOYMENT_STATUS_LABELS[status],
    dotClass: deploymentStatusColor(status),
    inFlight: statusGroupOf(status) === "building",
  };
}
