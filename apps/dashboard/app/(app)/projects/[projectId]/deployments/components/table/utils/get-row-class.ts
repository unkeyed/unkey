import type { Deployment } from "@/lib/collections";

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

export const FAILED_STATUS_STYLES = {
  base: "text-grayA-9 bg-error-1",
  hover: "hover:text-grayA-11 hover:bg-error-2",
  selected: "text-grayA-12 bg-error-3 hover:bg-error-3",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-error-7",
};

export const getRowClassName = (deployment: Deployment, selectedDeploymentId?: string) => {
  const isFailed = deployment.status === "failed";
  const style = isFailed ? FAILED_STATUS_STYLES : STATUS_STYLES;
  const isSelected = typeof selectedDeploymentId !== "undefined" && deployment.id === selectedDeploymentId;

  return cn(
    style.base,
    style.hover,
    "group rounded",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
