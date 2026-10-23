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
import { type StageKey, isPrebuilt } from "./details/stages-model";

export type DeploymentBody = "skipped" | "superseded" | "cancelled" | "stages";

export type BuildRequest = "none" | "seen" | "active";

export function buildRequestOf({
  inUrl,
  seenOnPage,
  stepsSettled,
}: {
  inUrl: boolean;
  seenOnPage: boolean;
  stepsSettled: boolean;
}): BuildRequest {
  if (inUrl && stepsSettled) {
    return "active";
  }
  return seenOnPage ? "seen" : "none";
}

export type DeploymentPageInput = {
  deployment: Pick<
    Deployment,
    "id" | "environmentId" | "instances" | "targetRegions" | "desiredState" | "source"
  >;
  status: DeploymentStatus;
  buildRequest: BuildRequest;
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
  buildRequest,
  domains,
  domainsLoading,
  customDomains,
  currentDeploymentId,
}: DeploymentPageInput): DeploymentPageModel {
  const instances = summarizeInstances(deployment);
  const hasBuildLogs = !isPrebuilt(deployment);
  const domainPriority = getDomainPriority({
    domains,
    customDomains,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    currentDeploymentId,
  });
  return {
    status,
    body:
      hasBuildLogs && buildRequest !== "none" ? BODY_WITH_BUILD[status] : BODY_BY_STATUS[status],
    requestedStage: hasBuildLogs && buildRequest === "active" ? "build" : null,
    instances,
    pageStatus: resolvePageStatus(status, instances),
    domains: domainPriority,
    domainCount: domainsLoading ? null : domainPriority.all.length,
  };
}

const BODY_BY_STATUS: Record<DeploymentStatus, DeploymentBody> = {
  pending: "stages",
  awaiting_approval: "stages",
  starting: "stages",
  building: "stages",
  deploying: "stages",
  network: "stages",
  finalizing: "stages",
  ready: "stages",
  failed: "stages",
  stopped: "stages",
  skipped: "skipped",
  superseded: "superseded",
  cancelled: "cancelled",
};

// A cancelled or superseded deployment may have built, so a build request shows
// its stages; a skipped one never did
const BODY_WITH_BUILD: Record<DeploymentStatus, DeploymentBody> = {
  ...BODY_BY_STATUS,
  superseded: "stages",
  cancelled: "stages",
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
