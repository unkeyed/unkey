export const SEVERITY_LEVELS = ["ERROR", "WARN", "INFO", "DEBUG"] as const;

export const SEVERITY_COLORS = {
  ERROR: "error",
  WARN: "warning",
  INFO: "info",
  DEBUG: "grayA",
} as const;

export const DEFAULT_LIMIT = 50;
export const DEFAULT_TIME_RANGE = "6h"; // 6 hours (matching backend default)
