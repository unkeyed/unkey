export const DEPLOYMENT_STATUSES = [
  "pending",
  "starting",
  "building",
  "deploying",
  "network",
  "finalizing",
  "ready",
  "failed",
] as const;

export type DeploymentStatus = (typeof DEPLOYMENT_STATUSES)[number];
