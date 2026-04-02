import { match } from "@unkey/match";
import type { DeploymentStatus } from "@/lib/collections";
import type { StepsData } from "./(deployment-progress)/deployment-progress";

const DEPLOYMENT_STATUSES: ReadonlySet<string> = new Set<DeploymentStatus>([
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "ready",
  "failed",
  "skipped",
  "awaiting_approval",
]);

function isDeploymentStatus(value: string): value is DeploymentStatus {
  return DEPLOYMENT_STATUSES.has(value);
}

const inProgress = (step: { endedAt: number | null } | null | undefined): boolean =>
  step != null && step.endedAt === null;

export function deriveStatusFromSteps(
  steps: StepsData | undefined,
  fallback: string,
): DeploymentStatus {
  return match(steps)
    .returnType<DeploymentStatus>()
    // awaiting_approval is authoritative from the DB — steps can't derive it
    .when(() => fallback === "awaiting_approval", () => "awaiting_approval")
    .when(
      (s) =>
        [s?.queued, s?.building, s?.deploying, s?.network, s?.finalizing, s?.starting].some(
          (step) => step?.error,
        ),
      () => "failed",
    )
    .when((s) => inProgress(s?.finalizing), () => "finalizing")
    .when((s) => Boolean(s?.finalizing?.completed), () => "ready")
    .when((s) => inProgress(s?.network), () => "network")
    .when((s) => inProgress(s?.deploying), () => "deploying")
    .when((s) => inProgress(s?.building), () => "building")
    .when((s) => inProgress(s?.starting), () => "starting")
    .when((s) => inProgress(s?.queued), () => "pending")
    .otherwise(() => (isDeploymentStatus(fallback) ? fallback : "pending"));
}
