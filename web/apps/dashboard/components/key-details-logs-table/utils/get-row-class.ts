import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { STATUS_STYLES, type StatusStyle } from "@unkey/ui";
import { cn } from "cn";

export const OUTCOME_STYLES = {
  success: STATUS_STYLES,
  warning: {
    base: "text-warningA-11 bg-warning-2",
    hover: "hover:text-warningA-11 hover:bg-warningA-3",
    selected: "text-warningA-11 bg-warningA-3",
    badge: {
      default: "bg-warningA-3 text-warningA-11 group-hover:bg-warningA-4",
      selected: "bg-warningA-4 text-warningA-11",
    },
    focusRing: "focus:ring-warning-7",
  },
  blocked: {
    base: "text-orangeA-11 bg-orange-2",
    hover: "hover:text-orangeA-11 hover:bg-orangeA-3",
    selected: "text-orangeA-11 bg-orangeA-3",
    badge: {
      default: "bg-orangeA-3 text-orangeA-11 group-hover:bg-orangeA-4",
      selected: "bg-orangeA-4 text-orangeA-11",
    },
    focusRing: "focus:ring-orange-7",
  },
  error: {
    base: "text-errorA-11 bg-error-2",
    hover: "hover:text-errorA-11 hover:bg-errorA-3",
    selected: "text-errorA-11 bg-errorA-3",
    badge: {
      default: "bg-errorA-3 text-errorA-11 group-hover:bg-errorA-4",
      selected: "bg-errorA-4 text-errorA-11",
    },
    focusRing: "focus:ring-error-7",
  },
} satisfies Record<"success" | "warning" | "blocked" | "error", StatusStyle>;

export const categorizeSeverity = (outcome: string): keyof typeof OUTCOME_STYLES => {
  switch (outcome) {
    case "VALID":
      return "success";
    case "RATE_LIMITED":
      return "warning";
    case "DISABLED":
    case "EXPIRED":
      return "blocked";
    case "INSUFFICIENT_PERMISSIONS":
    case "FORBIDDEN":
    case "USAGE_EXCEEDED":
      return "error";
    default:
      return "success";
  }
};

export const getStatusStyle = (log: KeyDetailsLog): StatusStyle => {
  const severity = categorizeSeverity(log.outcome);
  return OUTCOME_STYLES[severity];
};

export const getRowClassName = (log: KeyDetailsLog, selectedLog: KeyDetailsLog | null): string => {
  const style = getStatusStyle(log);
  const isSelected = selectedLog?.request_id === log.request_id;

  return cn(
    style.base,
    style.hover,
    "group rounded-md cursor-pointer transition-colors",
    "focus:outline-hidden focus:ring-1",
    style.focusRing,
    isSelected && style.selected,
  );
};
