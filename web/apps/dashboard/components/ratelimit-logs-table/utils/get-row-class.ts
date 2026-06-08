import { cn } from "@/lib/utils";
import { STATUS_STYLES, type StatusStyle } from "@unkey/ui";
import type { EnrichedRatelimitLog } from "../hooks/use-ratelimit-logs-query";

// A blocked ratelimit request is encoded as status === 0.
export const BLOCKED_STATUS = 0;

// Row and badge styles keyed by ratelimit decision severity: `success` for a
// passed request, `warning` for a blocked one. `success` reuses @unkey/ui's
// shared STATUS_STYLES so passed rows match every other selectable table;
// `warning` is the bespoke blocked-request variant with no shared equivalent.
// Module-private: callers go through getStatusStyle, not the map directly.
const RATELIMIT_ROW_STYLES = {
  success: STATUS_STYLES,
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
} satisfies Record<"success" | "warning", StatusStyle>;

export const getStatusStyle = (status: number): StatusStyle => {
  if (status === BLOCKED_STATUS) {
    return RATELIMIT_ROW_STYLES.warning;
  }
  return RATELIMIT_ROW_STYLES.success;
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
