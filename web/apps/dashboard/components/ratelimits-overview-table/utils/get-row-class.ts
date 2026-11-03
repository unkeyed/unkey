import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { STATUS_STYLES, type StatusStyle } from "@unkey/ui";
import { cn } from "cn";
import { isMostlyBlocked } from "./calculate-blocked-percentage";

export const RATELIMIT_OVERVIEW_STYLES = {
  success: STATUS_STYLES,
  blocked: {
    base: "text-orange-11",
    hover: "hover:bg-orange-3",
    selected: "text-orange-11 bg-orange-3",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-11 hover:bg-orange-5",
    },
    focusRing: "focus:ring-orange-7",
  },
} satisfies Record<"success" | "blocked", StatusStyle>;

export const getStatusStyle = (log: RatelimitOverviewLog): StatusStyle => {
  return isMostlyBlocked(log)
    ? RATELIMIT_OVERVIEW_STYLES.blocked
    : RATELIMIT_OVERVIEW_STYLES.success;
};

export const getRowClassName = (
  log: RatelimitOverviewLog,
  selectedLog: RatelimitOverviewLog | null,
) => {
  const hasMoreBlocked = isMostlyBlocked(log);
  const style = getStatusStyle(log);
  const isSelected = selectedLog?.request_id === log.request_id;

  return cn(
    style.base,
    style.hover,
    hasMoreBlocked ? "bg-orange-2" : "",
    "group rounded-md",
    "focus:outline-hidden focus:ring-1",
    style.focusRing,
    isSelected && style.selected,
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  );
};
