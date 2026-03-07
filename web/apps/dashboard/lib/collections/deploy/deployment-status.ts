export const DEPLOYMENT_STATUSES = [
  "awaiting_approval",
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "ready",
  "failed",
  "rejected",
] as const;

export type DeploymentStatus = (typeof DEPLOYMENT_STATUSES)[number];
