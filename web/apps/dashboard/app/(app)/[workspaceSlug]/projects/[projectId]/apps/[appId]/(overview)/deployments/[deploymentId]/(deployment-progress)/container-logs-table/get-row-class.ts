import { cn } from "@unkey/ui/src/lib/utils";
import type { ContainerLogRow } from "./columns";

const baseClasses = [
  "group",
  "rounded-none",
  "[&>td]:rounded-none",
  "focus:outline-hidden",
  "focus:ring-1",
  "focus:ring-opacity-40",
];

export function getContainerLogRowClass(log: ContainerLogRow): string {
  if (!log?.severity) {
    return "";
  }

  switch (log.severity.toUpperCase()) {
    case "ERROR":
      return cn(
        ...baseClasses,
        "text-error-11 bg-error-2",
        "hover:bg-error-3",
        "focus:ring-error-7",
      );
    case "WARN":
      return cn(
        ...baseClasses,
        "text-warning-11 bg-warning-2",
        "hover:bg-warning-3",
        "focus:ring-warning-7",
      );
    default:
      return cn(
        ...baseClasses,
        "text-grayA-9",
        "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
        "focus:ring-accent-7",
      );
  }
}
