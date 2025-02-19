export const DEFAULT_DRAGGABLE_WIDTH = 500;
export const MAX_DRAGGABLE_WIDTH = 800;
export const MIN_DRAGGABLE_WIDTH = 300;

export const YELLOW_STATES = ["RATE_LIMITED", "EXPIRED", "USAGE_EXCEEDED"];
export const RED_STATES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];

export const METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"] as const;
export const STATUSES = [200, 400, 500] as const;

export const HISTORICAL_DATA_WINDOW = 12 * 60 * 60 * 1000;
