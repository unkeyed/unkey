import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { getErrorSeverity } from "./calculate-blocked-percentage";

export type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: {
    default: string;
    selected: string;
  };
  focusRing: string;
};

export const STATUS_STYLES = {
  success: {
    base: "text-grayA-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
    selected: "text-accent-12 bg-grayA-3 hover:text-accent-12",
    badge: {
      default: "bg-gray-3 text-grayA-11 group-hover:bg-gray-5",
      selected: "bg-gray-5 text-grayA-12 hover:bg-gray-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warning-11 bg-warning-2",
    hover: "hover:text-warning-11 hover:bg-warning-3",
    selected: "text-warning-11 bg-warning-3",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-11 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
  },
  moderate: {
    base: "text-orange-11 bg-orange-2",
    hover: "hover:text-orange-11 hover:bg-orange-3",
    selected: "text-orange-11 bg-orange-3",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-11 hover:bg-orange-5",
    },
    focusRing: "focus:ring-orange-7",
  },
  error: {
    base: "text-error-11 bg-error-2",
    hover: "hover:text-error-11 hover:bg-error-3",
    selected: "text-error-11 bg-error-3",
    badge: {
      default: "bg-error-4 text-error-11 group-hover:bg-error-5",
      selected: "bg-error-5 text-error-11 hover:bg-error-5",
    },
    focusRing: "focus:ring-error-7",
  },
};

export const categorizeSeverity = (outcome: string): keyof typeof STATUS_STYLES => {
  switch (outcome) {
    // Critical errors
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
      return "error";

    // Moderate errors
    case "RATE_LIMITED":
    case "USAGE_EXCEEDED":
      return "moderate";

    // Warnings
    case "DISABLED":
    case "EXPIRED":
      return "warning";

    default:
      return "success";
  }
};

// Get status style based on error severity
export const getStatusStyle = (log: KeysOverviewLog): StatusStyle => {
  const severity = getErrorSeverity(log);

  switch (severity) {
    case "high":
      return STATUS_STYLES.error;
    case "moderate":
      return STATUS_STYLES.moderate;
    case "low":
      return STATUS_STYLES.warning;
    default:
      return STATUS_STYLES.success;
  }
};

export const getOutcomeBadgeClass = (outcome: string): string => {
  const severity = categorizeSeverity(outcome);

  switch (severity) {
    case "error":
      return "bg-error-4 text-error-11";
    case "moderate":
      return "bg-orange-4 text-orange-11";
    case "warning":
      return "bg-warning-4 text-warning-11";
    case "success":
      return "bg-accent-4 text-accent-11";
    default:
      return "bg-gray-4 text-gray-11";
  }
};

export const getRowClassName = (log: KeysOverviewLog, selectedLog: KeysOverviewLog) => {
  const style = getStatusStyle(log);
  const isSelected = log.key_id === selectedLog?.key_id;

  return cn(
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  );
};
