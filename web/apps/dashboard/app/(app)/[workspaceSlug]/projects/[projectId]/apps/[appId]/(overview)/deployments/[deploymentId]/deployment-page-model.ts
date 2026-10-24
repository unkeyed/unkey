import type { CustomDomain, Domain } from "@/lib/collections";
import {
  DEPLOYMENT_GROUP_COLOR,
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
  deploymentStatusColor,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import { type Deployment, hasBuildLogs } from "@/lib/collections/deploy/deployments";
import {
  type InstanceSummary,
  instanceLabel,
  summarizeInstances,
} from "@/lib/collections/deploy/instance-status";
import { type DomainPriorityResult, getDomainPriority } from "../../../components/domain-priority";
import type { StageKey } from "./details/stages-model";

export type DeploymentBody = "skipped" | "stages";

export type DeploymentPageInput = {
  deployment: Pick<
    Deployment,
    "id" | "environmentId" | "instances" | "targetRegions" | "desiredState" | "source"
  >;
  status: DeploymentStatus;
  buildRequested: boolean;
  domains: ReadonlyArray<Domain>;
  domainsLoading: boolean;
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
  requestedStage: StageKey | null;
  instances: InstanceSummary | null;
  pageStatus: PageStatus;
  domains: DomainPriorityResult;
  domainCount: number | null;
};

export function deploymentPageModel({
  deployment,
  status,
  buildRequested,
  domains,
  domainsLoading,
  customDomains,
  currentDeploymentId,
}: DeploymentPageInput): DeploymentPageModel {
  const instances = summarizeInstances(deployment);
  const domainPriority = getDomainPriority({
    domains,
    customDomains,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    currentDeploymentId,
  });
  return {
    status,
    body: status === "skipped" ? "skipped" : "stages",
    requestedStage:
      buildRequested && hasBuildLogs({ source: deployment.source, status }) ? "build" : null,
    instances,
    pageStatus: resolvePageStatus(status, instances),
    domains: domainPriority,
    domainCount: domainsLoading ? null : domainPriority.all.length,
  };
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
