export const INSTANCE_STATUSES = ["inactive", "pending", "running", "failed"] as const;

export type InstanceStatus = (typeof INSTANCE_STATUSES)[number];
