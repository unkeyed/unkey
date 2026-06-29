export const DEPLOYMENT_STATUSES = [
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
] as const;

export type DeploymentStatus = (typeof DEPLOYMENT_STATUSES)[number];

// Statuses where the deployment has settled and won't change on its own. A
// deployment in any other status is still progressing, so consumers poll for
// updates while one is in flight.
const TERMINAL_DEPLOYMENT_STATUSES = new Set<DeploymentStatus>([
  "ready",
  "failed",
  "skipped",
  "stopped",
  "superseded",
  "cancelled",
]);

export function isDeploymentInFlight(status: DeploymentStatus): boolean {
  return !TERMINAL_DEPLOYMENT_STATUSES.has(status);
}
