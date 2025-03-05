import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { calculateErrorPercentage } from "./calculate-blocked-percentage";

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
    base: "text-accent-9",
    hover: "hover:bg-accent-3",
    selected: "text-accent-11 bg-accent-3",
    badge: {
      default: "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
      selected: "bg-accent-5 text-accent-12 hover:bg-hover-5",
    },
    focusRing: "focus:ring-accent-7",
  },
  error: {
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

export const getStatusStyle = (log: KeysOverviewLog): StatusStyle => {
  return calculateErrorPercentage(log)
    ? STATUS_STYLES.error
    : STATUS_STYLES.success;
};

export const getRowClassName = (log: KeysOverviewLog) => {
  const hasHighErrorRate = calculateErrorPercentage(log);
  const style = getStatusStyle(log);

  return cn(
    style.base,
    style.hover,
    hasHighErrorRate ? "bg-orange-2" : "",
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing
  );
};
