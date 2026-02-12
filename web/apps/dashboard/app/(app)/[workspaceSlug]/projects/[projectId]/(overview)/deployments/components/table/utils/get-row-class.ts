import type { Deployment, Environment } from "@/lib/collections";

import { cn } from "@/lib/utils";

export type StatusStyle = {
  base: string;
  hover: string;
  badge: {
    default: string;
  };
  focusRing: string;
};

export const STATUS_STYLES = {
  base: "text-grayA-9",
  hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-2",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
  },
  focusRing: "focus:ring-accent-7",
};

export const FAILED_STATUS_STYLES = {
  base: "text-grayA-9 bg-error-2",
  hover: "hover:text-grayA-11 hover:bg-error-2",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-error-7",
};

export const ROLLED_BACK_STYLES = {
  base: "text-grayA-9 bg-warning-2",
  hover: "hover:text-grayA-11 hover:bg-warning-4",
  badge: {
    default: "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5 border-transparent",
    selected: "bg-grayA-5 text-grayA-12 hover:bg-grayA-5 border-grayA-3",
  },
  focusRing: "focus:ring-warning-7",
};

export const getRowClassName = (
  { deployment }: { deployment: Deployment; environment: Environment },
  liveDeploymentId: string | null,
  isRolledBack: boolean,
) => {
  const isFailed = deployment.status === "failed";

  const style = isFailed
    ? FAILED_STATUS_STYLES
    : isRolledBack && liveDeploymentId === deployment.id
      ? ROLLED_BACK_STYLES
      : STATUS_STYLES;

  return cn(
    style.base,
    style.hover,
    "group rounded",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
  );
};
