import {
  type DeploymentStatusGroup,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import type { ProjectApp } from "@/lib/collections/deploy/project-cards";
import { match } from "@unkey/match";
import { useElapsed } from "@unkey/ui";
import { cn } from "cn";
import { DeploymentStatusIndicator } from "../../[projectId]/apps/[appId]/components/deployment-status-dot";

export type AppDeployment = NonNullable<ProjectApp["headlineDeployment"]>;

const VERB: Record<DeploymentStatusGroup, string> = {
  queued: "started",
  building: "started",
  failed: "failed",
  blocked: "awaiting approval",
  stopped: "stopped",
  cancelled: "cancelled",
  skipped: "skipped",
  ready: "deployed",
  superseded: "deployed",
};

const TONE: Record<DeploymentStatusGroup, string> = {
  queued: "text-gray-9",
  building: "text-gray-9",
  failed: "text-error-11",
  blocked: "text-warning-11",
  stopped: "text-gray-9",
  cancelled: "text-gray-9",
  skipped: "text-gray-9",
  ready: "text-gray-9",
  superseded: "text-gray-9",
};

export function useDeploymentPhrase(deployment: AppDeployment): string {
  const deployedAgo = useElapsed(deployment.deployedAt, "narrow");
  return `${VERB[statusGroupOf(deployment.status)]} ${deployedAgo}`;
}

export function DeploymentMeta({ deployment }: { deployment: AppDeployment }) {
  const deployedAgo = useElapsed(deployment.deployedAt, "narrow");

  const settled = (group: DeploymentStatusGroup) => (
    <span className={cn("shrink-0 text-xs", TONE[group])}>
      {VERB[group]} {deployedAgo}
    </span>
  );

  return match(statusGroupOf(deployment.status))
    .with("queued", "building", (group) => (
      <span className="flex shrink-0 items-center gap-1.5 text-xs text-gray-9">
        <DeploymentStatusIndicator status={deployment.status} />
        <span className="sr-only">{VERB[group]}</span>
        {deployedAgo}
      </span>
    ))
    .with("ready", "superseded", (group) => (
      <span className={cn("shrink-0 text-xs", TONE[group])}>{deployedAgo}</span>
    ))
    .with("failed", "blocked", settled)
    .with("stopped", "cancelled", settled)
    .with("skipped", settled)
    .exhaustive();
}
