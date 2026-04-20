import type { DeploymentStatus } from "@/lib/collections";
import { match } from "@unkey/match";
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
  "stopped",
  "superseded",
  "cancelled",
]);

// Statuses that are authoritative from the DB and cannot be derived from
// step state. Cancelled/superseded steps do carry an `error` marker, so
// without this short-circuit deriveStatusFromSteps would (incorrectly)
// classify them as failed.
const AUTHORITATIVE_STATUSES: ReadonlySet<string> = new Set<DeploymentStatus>([
  "awaiting_approval",
  "cancelled",
  "superseded",
  "stopped",
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
    .when(
      () => AUTHORITATIVE_STATUSES.has(fallback),
      () => fallback as DeploymentStatus,
    )
    .when(
      (s) =>
        [s?.queued, s?.building, s?.deploying, s?.network, s?.finalizing, s?.starting].some(
          (step) => step?.error,
        ),
      () => "failed",
    )
    .when(
      (s) => inProgress(s?.finalizing),
      () => "finalizing",
    )
    .when(
      (s) => Boolean(s?.finalizing?.completed),
      () => "ready",
    )
    .when(
      (s) => inProgress(s?.network),
      () => "network",
    )
    .when(
      (s) => inProgress(s?.deploying),
      () => "deploying",
    )
    .when(
      (s) => inProgress(s?.building),
      () => "building",
    )
    .when(
      (s) => inProgress(s?.starting),
      () => "starting",
    )
    .when(
      (s) => inProgress(s?.queued),
      () => "pending",
    )
    .otherwise(() => (isDeploymentStatus(fallback) ? fallback : "pending"));
}
