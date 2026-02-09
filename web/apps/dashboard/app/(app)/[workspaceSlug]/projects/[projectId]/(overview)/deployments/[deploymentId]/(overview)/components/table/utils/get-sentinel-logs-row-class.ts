import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { cn } from "@unkey/ui/src/lib/utils";

type StatusStyle = {
  base: string;
  hover: string;
  badge: {
    default: string;
    selected: string;
  };
  focusRing: string;
};

const STATUS_STYLES = {
  success: {
    base: "text-grayA-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
    badge: {
      default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5",
      selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  warning: {
    base: "text-warning-11 bg-warning-2",
    hover: "hover:bg-warning-3",
    badge: {
      default: "bg-warning-4 text-warning-11 group-hover:bg-warning-5",
      selected: "bg-warning-5 text-warning-11 hover:bg-warning-5",
    },
    focusRing: "focus:ring-warning-7",
  },
  error: {
    base: "text-error-11 bg-error-2",
    hover: "hover:bg-error-3",
    badge: {
      default: "bg-error-4 text-error-11 group-hover:bg-error-5",
      selected: "bg-error-5 text-error-11 hover:bg-error-5",
    },
    focusRing: "focus:ring-error-7",
  },
};

export const getStatusStyle = (status: number): StatusStyle => {
  if (status >= 500) {
    return STATUS_STYLES.error;
  }
  if (status >= 400) {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

export const WARNING_ICON_STYLES = {
  base: "size-[13px] mb-[1px]",
  warning: "text-warning-11",
  error: "text-error-11",
};

export const getRowClassName = (log: SentinelLogsResponse): string => {
  // Early validation
  if (!log?.request_id) {
    throw new Error("Log must have a valid request_id");
  }

  if (
    !Number.isInteger(log.response_status) ||
    log.response_status < 100 ||
    log.response_status > 599
  ) {
    throw new Error(
      `Invalid response_status: ${log.response_status}. Must be a valid HTTP status code.`,
    );
  }

  const style = getStatusStyle(log.response_status);

  const baseClasses = [
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
  ];
  return cn(...baseClasses);
};
