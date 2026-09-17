"use client";

import { useNow } from "@/hooks/use-now";
import {
  type DeploymentStatusGroup,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import type { ProjectApp } from "@/lib/collections/deploy/projects";
import { cn } from "@/lib/utils";
import { match } from "@unkey/match";
import { intlFormatDistance } from "date-fns";
import { DeploymentStatusIndicator } from "../../[projectId]/apps/[appId]/components/deployment-status-dot";

export type AppDeployment = NonNullable<ProjectApp["headlineDeployment"]>;

const VERB: Record<DeploymentStatusGroup, string> = {
  queued: "started",
  building: "started",
  failed: "failed",
  blocked: "awaiting approval",
  stopped: "stopped",
  cancelled: "cancelled",
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
  ready: "text-gray-9",
  superseded: "text-gray-9",
};

// deployedAt is the server's clock read against the browser's, so a browser
// running behind would put the deployment in the future: "started in 25 sec".
function useDeploymentAge(deployment: AppDeployment): string {
  const now = useNow();
  return intlFormatDistance(deployment.deployedAt, Math.max(now, deployment.deployedAt), {
    style: "narrow",
  });
}

export function useDeploymentPhrase(deployment: AppDeployment): string {
  return `${VERB[statusGroupOf(deployment.status)]} ${useDeploymentAge(deployment)}`;
}

export function DeploymentMeta({ deployment }: { deployment: AppDeployment }) {
  const deployedAgo = useDeploymentAge(deployment);
  const deployedPhrase = useDeploymentPhrase(deployment);

  const settled = (group: DeploymentStatusGroup) => (
    <span className={cn("shrink-0 text-xs", TONE[group])}>{deployedPhrase}</span>
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
    .exhaustive();
}
