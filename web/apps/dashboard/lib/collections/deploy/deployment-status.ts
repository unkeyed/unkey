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

export const DEPLOYMENT_STATUS_LABELS: Record<DeploymentStatus, string> = {
  pending: "Pending",
  starting: "Starting",
  building: "Building",
  deploying: "Deploying",
  network: "Assigning Domains",
  finalizing: "Finalizing",
  ready: "Ready",
  failed: "Failed",
  skipped: "Skipped",
  awaiting_approval: "Awaiting Approval",
  stopped: "Stopped",
  superseded: "Superseded",
  cancelled: "Cancelled",
};

// Statuses where the deployment has settled and won't change on its own. A
// deployment in any other status is still progressing, so consumers poll for
// updates while one is in flight.
const TERMINAL_DEPLOYMENT_STATUSES = new Set<DeploymentStatus>([
  "ready",
  "failed",
  "skipped",
  "awaiting_approval",
  "stopped",
  "superseded",
  "cancelled",
]);

export function isDeploymentInFlight(status: DeploymentStatus): boolean {
  return !TERMINAL_DEPLOYMENT_STATUSES.has(status);
}

// The filter groups a user picks from. The pipeline's intermediate statuses
// (starting, network, finalizing, ...) read as one "building" phase from the
// outside, and skipped rows are cancellations the platform made on the user's
// behalf. Every raw status belongs to exactly one group; the test enforces it.
export const DEPLOYMENT_STATUS_GROUPS = {
  ready: ["ready"],
  failed: ["failed"],
  building: ["starting", "building", "deploying", "network", "finalizing"],
  queued: ["pending"],
  blocked: ["awaiting_approval"],
  cancelled: ["cancelled", "skipped"],
  superseded: ["superseded"],
  stopped: ["stopped"],
} as const satisfies Record<string, readonly DeploymentStatus[]>;

export type DeploymentStatusGroup = keyof typeof DEPLOYMENT_STATUS_GROUPS;

export const DEPLOYMENT_STATUS_GROUP_NAMES = Object.keys(
  DEPLOYMENT_STATUS_GROUPS,
) as DeploymentStatusGroup[];

export function isDeploymentStatusGroup(value: string): value is DeploymentStatusGroup {
  return Object.hasOwn(DEPLOYMENT_STATUS_GROUPS, value);
}

export function expandDeploymentStatusGroups(
  groups: readonly DeploymentStatusGroup[],
): DeploymentStatus[] {
  return groups.flatMap((group) => DEPLOYMENT_STATUS_GROUPS[group]);
}
