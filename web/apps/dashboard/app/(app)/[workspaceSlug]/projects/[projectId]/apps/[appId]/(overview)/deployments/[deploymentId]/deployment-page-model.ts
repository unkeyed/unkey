import type { CustomDomain, Domain } from "@/lib/collections";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { type InstanceSummary, summarizeInstances } from "@/lib/collections/deploy/instance-status";
import { type DomainPriorityResult, getDomainPriority } from "../../../components/domain-priority";
import { type PageStatus, resolvePageStatus } from "./details/deployment-health";

export type DeploymentBody = "build" | "skipped" | "superseded" | "cancelled" | "stages";

export type DeploymentPageInput = {
  deployment: Pick<Deployment, "id" | "environmentId" | "instances" | "desiredInstanceCount">;
  status: DeploymentStatus;
  showBuild: boolean;
  domains: ReadonlyArray<Domain>;
  customDomains: ReadonlyArray<CustomDomain>;
  currentDeploymentId: string | null;
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
