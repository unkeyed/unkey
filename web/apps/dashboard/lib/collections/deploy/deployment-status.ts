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
] as const;

export type DeploymentStatus = (typeof DEPLOYMENT_STATUSES)[number];
