import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import type { StatusStyle } from "@unkey/ui";
import { getErrorSeverity } from "./calculate-blocked-percentage";

export const SEVERITY_STYLES = {
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

// Get status style based on error severity
export const getStatusStyle = (log: KeysOverviewLog): StatusStyle => {
  const severity = getErrorSeverity(log);

  switch (severity) {
    case "high":
      return SEVERITY_STYLES.error;
    case "moderate":
      return SEVERITY_STYLES.moderate;
    case "low":
      return SEVERITY_STYLES.warning;
    default:
      return SEVERITY_STYLES.success;
  }
};

export const getRowClassName = (log: KeysOverviewLog, selectedLog: KeysOverviewLog | null) => {
  const style = getStatusStyle(log);
  const isSelected = log.key_id === selectedLog?.key_id;

  return cn(
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  );
};
