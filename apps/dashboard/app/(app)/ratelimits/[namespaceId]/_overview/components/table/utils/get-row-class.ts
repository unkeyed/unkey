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
  focusRing: string;
};

export const STATUS_STYLES = {
  success: {
    base: "text-accent-9",
    hover: "hover:bg-accent-3",
    selected: "text-accent-11 bg-accent-3",
    badge: {
      default: "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
      selected: "bg-accent-5 text-accent-12 hover:bg-hover-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  blocked: {
    base: "text-orange-11",
    hover: "hover:bg-orange-3",
    selected: "bg-orange-3",
    badge: {
      default: "bg-orange-4 text-orange-11 group-hover:bg-orange-5",
      selected: "bg-orange-5 text-orange-11 hover:bg-orange-5",
    },
    focusRing: "focus:ring-orange-7",
  },
};

export const getStatusStyle = (log: RatelimitOverviewLog): StatusStyle => {
  return calculateBlockedPercentage(log) ? STATUS_STYLES.blocked : STATUS_STYLES.success;
};

export const getRowClassName = (log: RatelimitOverviewLog) => {
  const hasMoreBlocked = calculateBlockedPercentage(log);
  const style = getStatusStyle(log);

  return cn(
    style.base,
    style.hover,
    hasMoreBlocked ? "bg-orange-2" : "",
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
  );
};
