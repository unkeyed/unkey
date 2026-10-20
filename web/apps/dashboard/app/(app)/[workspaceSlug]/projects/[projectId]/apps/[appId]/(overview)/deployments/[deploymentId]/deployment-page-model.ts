import type { CustomDomain, Domain } from "@/lib/collections";
import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
  deploymentStatusColor,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import {
  type InstanceSummary,
  instanceLabel,
  summarizeInstances,
} from "@/lib/collections/deploy/instance-status";
import { type DomainPriorityResult, getDomainPriority } from "../../../components/domain-priority";

export type DeploymentBody = "build" | "skipped" | "superseded" | "cancelled" | "stages";

export type DeploymentPageInput = {
  deployment: Pick<
    Deployment,
    "id" | "environmentId" | "instances" | "targetRegions" | "desiredState"
  >;
  status: DeploymentStatus;
  showBuild: boolean;
  domains: ReadonlyArray<Domain>;
  customDomains: ReadonlyArray<CustomDomain>;
  currentDeploymentId: string | null;
};

export type PageStatus = {
  label: string;
  dotClass: string;
  inFlight: boolean;
};

export type DeploymentPageModel = {
  status: DeploymentStatus;
  body: DeploymentBody;
  instances: InstanceSummary | null;
  pageStatus: PageStatus;
  domains: DomainPriorityResult;
};

export function deploymentPageModel({
  deployment,
  status,
  showBuild,
  domains,
  customDomains,
  currentDeploymentId,
}: DeploymentPageInput): DeploymentPageModel {
  const instances = summarizeInstances(deployment);
  return {
    status,
    body: showBuild ? "build" : bodyOf(status),
    instances,
    pageStatus: resolvePageStatus(status, instances),
    domains: getDomainPriority({
      domains,
      customDomains,
      environmentId: deployment.environmentId,
      deploymentId: deployment.id,
      currentDeploymentId,
    }),
  };
}

function bodyOf(status: DeploymentStatus): DeploymentBody {
  switch (status) {
    case "skipped":
    case "superseded":
    case "cancelled":
      return status;
    default:
      return "stages";
  }
}

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
