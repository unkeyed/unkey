import { cn } from "@/lib/utils";
import type { EnrichedRatelimitLog } from "../hooks/use-ratelimit-logs-query";

// A blocked ratelimit request is encoded as status === 0.
export const BLOCKED_STATUS = 0;

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
      default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5",
      selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warning-11 bg-warning-2",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-11 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
  },
};

export const getStatusStyle = (status: number): StatusStyle => {
  if (status === BLOCKED_STATUS) {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

export const getRowClassName = (
  log: EnrichedRatelimitLog,
  selectedLog: EnrichedRatelimitLog | null,
): string => {
  const style = getStatusStyle(log.status);
  const isSelected = selectedLog?.request_id === log.request_id;

  return cn(
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
