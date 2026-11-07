import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import { STATUS_STYLES, type StatusStyle } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

const RESPONSE_STATUS_STYLES = {
  success: STATUS_STYLES,
  info: {
    base: "text-info-11 bg-info-2",
    hover: "hover:bg-info-3",
    selected: "bg-info-3",
    badge: {
      default: "bg-info-4 text-info-11 group-hover:bg-info-5",
      selected: "bg-info-5 text-info-11 hover:bg-info-5",
    },
    focusRing: "focus:ring-info-7",
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
  error: {
    base: "text-error-11 bg-error-2",
    hover: "hover:bg-error-3",
    selected: "bg-error-3",
    badge: {
      default: "bg-error-4 text-error-11 group-hover:bg-error-5",
      selected: "bg-error-5 text-error-11 hover:bg-error-5",
    },
    focusRing: "focus:ring-error-7",
  },
} satisfies Record<"success" | "info" | "warning" | "error", StatusStyle>;

export const getStatusStyle = (status: number): StatusStyle => {
  if (!Number.isInteger(status) || status < 100 || status >= 500) {
    return RESPONSE_STATUS_STYLES.error;
  }
  if (status >= 400) {
    return RESPONSE_STATUS_STYLES.warning;
  }
  if (status >= 300) {
    return RESPONSE_STATUS_STYLES.info;
  }
  return RESPONSE_STATUS_STYLES.success;
};

export const WARNING_ICON_STYLES = {
  base: "size-[13px] mb-px",
  warning: "text-warning-11",
  error: "text-error-11",
};

export const getSelectedClassName = (log: RequestLogsResponse, isSelected: boolean) => {
  if (!isSelected) {
    return "";
  }
  const style = getStatusStyle(log.response_status);
  return style.selected;
};

export const getRowClassName = (
  log: RequestLogsResponse,
  selectedLog?: RequestLogsResponse | null,
  isLive?: boolean,
  realtimeLogs?: RequestLogsResponse[],
): string => {
  if (!log?.request_id) {
    throw new Error("Log must have a valid request_id");
  }

  const style = getStatusStyle(log.response_status);
  const isSelected = Boolean(selectedLog?.request_id === log.request_id);

  const baseClasses = [
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-hidden focus:ring-1",
    style.focusRing,
  ];

  const conditionalClasses = [
    // Selected state
    isSelected && style.selected,

    // Live mode opacity for historical logs
    isLive &&
      realtimeLogs &&
      !realtimeLogs.some((rt) => rt.request_id === log.request_id) && [
        "opacity-50",
        "hover:opacity-100",
      ],

    // Selection-based z-index and opacity
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  ].filter(Boolean);

  return cn(...baseClasses, ...conditionalClasses);
};
