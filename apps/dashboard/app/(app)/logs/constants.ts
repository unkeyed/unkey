export const DEFAULT_DRAGGABLE_WIDTH = 500;

export const YELLOW_STATES = ["RATE_LIMITED", "EXPIRED", "USAGE_EXCEEDED"];
export const RED_STATES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];

export const METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"] as const;
export const STATUSES = [200, 400, 500] as const;
