import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type { ProjectApp } from "@/lib/collections/deploy/projects";
import { cn } from "@/lib/utils";
import { match } from "@unkey/match";
import { intlFormatDistance } from "date-fns";
import { DeploymentStatusIndicator } from "../../[projectId]/apps/[appId]/components/deployment-status-dot";

type AppDeployment = NonNullable<ProjectApp["headlineDeployment"]>;

type DeploymentPhase =
  | "in-flight"
  | "failed"
  | "awaiting-approval"
  | "stopped"
  | "cancelled"
  | "deployed";

function deploymentPhase(status: DeploymentStatus): DeploymentPhase {
  return match(status)
    .returnType<DeploymentPhase>()
    .with("pending", "starting", () => "in-flight")
    .with("building", "deploying", () => "in-flight")
    .with("network", "finalizing", () => "in-flight")
    .with("failed", () => "failed")
    .with("awaiting_approval", () => "awaiting-approval")
    .with("stopped", () => "stopped")
    .with("cancelled", "skipped", () => "cancelled")
    .with("ready", "superseded", () => "deployed")
    .exhaustive();
}

const VERB: Record<DeploymentPhase, string> = {
  "in-flight": "started",
  failed: "failed",
  "awaiting-approval": "awaiting approval",
  stopped: "stopped",
  cancelled: "cancelled",
  deployed: "deployed",
};

const TONE: Record<DeploymentPhase, string> = {
  "in-flight": "text-gray-9",
  failed: "text-error-11",
  "awaiting-approval": "text-warning-11",
  stopped: "text-gray-9",
  cancelled: "text-gray-9",
  deployed: "text-gray-9",
};

function age(deployedAt: number): string {
  return intlFormatDistance(deployedAt, Date.now(), { style: "narrow" });
}

export function deploymentPhrase(deployment: AppDeployment): string {
  return `${VERB[deploymentPhase(deployment.status)]} ${age(deployment.deployedAt)}`;
}

export function DeploymentMeta({ deployment }: { deployment: AppDeployment | null }) {
  if (!deployment) {
    return null;
  }

  const deployedAgo = age(deployment.deployedAt);

  const settled = (phase: DeploymentPhase) => (
    <span className={cn("shrink-0 text-xs", TONE[phase])}>
      {VERB[phase]} {deployedAgo}
    </span>
  );

  return match(deploymentPhase(deployment.status))
    .with("in-flight", (phase) => (
      <span className="flex shrink-0 items-center gap-1.5 text-xs text-gray-9">
        <DeploymentStatusIndicator status={deployment.status} />
        <span className="sr-only">{VERB[phase]}</span>
        {deployedAgo}
      </span>
    ))
    .with("deployed", (phase) => (
      <span className={cn("shrink-0 text-xs", TONE[phase])}>{deployedAgo}</span>
    ))
    .with("failed", "awaiting-approval", settled)
    .with("stopped", "cancelled", settled)
    .exhaustive();
}
