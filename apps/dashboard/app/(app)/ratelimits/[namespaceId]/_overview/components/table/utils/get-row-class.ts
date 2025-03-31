import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { calculateBlockedPercentage } from "./calculate-blocked-percentage";

type StatusStyle = {
  base: string;
  hover: string;
  selected: string;
  badge: {
    default: string;
    selected: string;
  };
  focus: string;
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
    focus: "focus:ring-accent-7",
  },
  blocked: {
    base: "text-orange-11",
    hover: "hover:bg-orange-3",
    selected: "text-orange-11 bg-orange-3",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-11 hover:bg-orange-5",
    },
    focus: "focus:ring-orange-7",
  },
};

export const getStatusStyle = (log: RatelimitOverviewLog): StatusStyle => {
  return calculateBlockedPercentage(log) ? STATUS_STYLES.blocked : STATUS_STYLES.success;
};

export const getRowClassName = (log: RatelimitOverviewLog, selectedLog: RatelimitOverviewLog) => {
  const hasMoreBlocked = calculateBlockedPercentage(log);
  const style = getStatusStyle(log);
  const isSelected = selectedLog?.request_id === log.request_id;

  return cn(
    style.base,
    style.hover,
    hasMoreBlocked ? "bg-orange-2" : "",
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focus,
    isSelected && style.selected,
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  );
};
