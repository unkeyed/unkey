export const DEPLOYMENT_STATUSES = [
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "ready",
  "failed",
  "awaiting_approval",
] as const;

export type DeploymentStatus = (typeof DEPLOYMENT_STATUSES)[number];
