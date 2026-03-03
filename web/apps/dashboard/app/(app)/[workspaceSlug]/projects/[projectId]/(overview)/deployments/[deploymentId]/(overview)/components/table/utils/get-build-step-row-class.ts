import { cn } from "@unkey/ui/src/lib/utils";
import type { BuildStepRow } from "../columns/build-steps";

export function getBuildStepRowClass(step: BuildStepRow): string {
  if (!step?.step_id) {
    return "";
  }

  const baseClasses = [
    "group",
    "rounded-md",
    "focus:outline-hidden",
    "focus:ring-1",
    "focus:ring-opacity-40",
  ];

  if (step.error) {
    return cn(...baseClasses, "text-error-11 bg-error-2", "hover:bg-error-3", "focus:ring-error-7");
  }

  if (step.cached) {
    return cn(...baseClasses, "text-blue-11 bg-blue-2", "hover:bg-blue-3", "focus:ring-blue-7");
  }

  return cn(
    ...baseClasses,
    "text-grayA-9",
    "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-grayA-3",
    "focus:ring-accent-7",
  );
}
