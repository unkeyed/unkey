import type { RuntimeLog } from "@/lib/schemas/runtime-logs.schema";
import { cn } from "@/lib/utils";

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

const STATUS_STYLES: Record<"success" | "warning" | "error", StatusStyle> = {
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
};

export const getSeverityStyle = (severity: string): StatusStyle => {
  const upper = severity.toUpperCase();
  if (upper === "ERROR") {
    return STATUS_STYLES.error;
  }
  if (upper === "WARN" || upper === "WARNING") {
    return STATUS_STYLES.warning;
  }
  return STATUS_STYLES.success;
};

// Content identity of a log line (runtime logs have no server id); drives
// realtime dedup and selection. Not collision-proof on its own, so the table
// derives unique React keys via RuntimeLogRow.
export const getLogKey = (log: RuntimeLog): string =>
  `${log.time}-${log.region}-${log.instance_id}-${log.message}`;

// A log row plus a unique render key for DataTable getRowId. Optional so a bare
// RuntimeLog (the selected log from context) stays assignable; getRowId falls
// back to getLogKey when absent.
export type RuntimeLogRow = RuntimeLog & { rowKey?: string };

// Suffix duplicate getLogKey values so byte-identical lines get a unique `rowKey`
// for DataTable getRowId. The first occurrence keeps the bare key so realtime and
// historical rows for the same line still match during the table's merge dedup.
export const attachRowKeys = (logs: RuntimeLog[]): RuntimeLogRow[] => {
  const counts = new Map<string, number>();
  return logs.map((log) => {
    const base = getLogKey(log);
    const occurrence = counts.get(base) ?? 0;
    counts.set(base, occurrence + 1);
    return { ...log, rowKey: occurrence === 0 ? base : `${base}#${occurrence}` };
  });
};

export const getSelectedClassName = (log: RuntimeLog, isSelected: boolean): string => {
  if (!isSelected) {
    return "";
  }
  return getSeverityStyle(log.severity).selected;
};

export const getRowClassName = (
  log: RuntimeLog,
  selectedLog?: RuntimeLog | null,
  isLive?: boolean,
  realtimeLogs?: RuntimeLog[],
): string => {
  const style = getSeverityStyle(log.severity);
  const logKey = getLogKey(log);
  const isSelected = selectedLog ? getLogKey(selectedLog) === logKey : false;

  const baseClasses = [
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
  ];

  const conditionalClasses = [
    // Selected state
    isSelected && style.selected,

    // Live mode dims historical logs so the realtime rows stand out.
    isLive &&
      realtimeLogs &&
      !realtimeLogs.some((rt) => getLogKey(rt) === logKey) && ["opacity-50", "hover:opacity-100"],

    // Selection-based z-index and opacity
    selectedLog && {
      "opacity-50 z-0": !isSelected,
      "opacity-100 z-10": isSelected,
    },
  ].filter(Boolean);

  return cn(...baseClasses, ...conditionalClasses);
};
