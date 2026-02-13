import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { cn } from "@/lib/utils";

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
  base: "text-grayA-9",
  hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-2",
  selected: "text-accent-12 bg-grayA-2 hover:text-accent-12",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-accent-7",
};

export const getRowClassName = (log: Permission, selectedLog: Permission | null) => {
  const style = STATUS_STYLES;
  const isSelected = log.permissionId === selectedLog?.permissionId;

  return cn(
    style.base,
    style.hover,
    "group rounded-sm",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
