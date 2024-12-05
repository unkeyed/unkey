import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

import type { Workspace } from "@/lib/db";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
export const isBrowser = typeof window !== "undefined";

export function debounce<T extends (...args: any[]) => any>(func: T, delay: number) {
  let timeoutId: ReturnType<typeof setTimeout>;

  function debounced(...args: Parameters<T>) {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => {
      func(...args);
    }, delay);
  }

  debounced.cancel = () => {
    clearTimeout(timeoutId);
  };

  return debounced;
}

type WorkspaceFeatures = Pick<Workspace, "features" | "betaFeatures">;

type ConfigObject = WorkspaceFeatures["betaFeatures"] & WorkspaceFeatures["features"];

type FlagValue<T extends keyof ConfigObject> = ConfigObject[T];

/**
 * Checks if a workspace has access to a specific feature or beta feature
 * Returns the feature value if access is granted, undefined otherwise
 * Note: Always returns true in development environment
 *
 * @param flagName - The name of the feature to check
 * @param workspace - The workspace to check access for
 * @returns The feature value (boolean | number | string) if access granted, undefined otherwise
 *
 * @example
 * ```typescript
 * // Check if workspace has access to logs page
 * if (!flag("logsPage", workspace)) {
 *   return notFound();
 * }
 *
 * // Check if workspace has access to a feature with numeric value
 * const userLimit = flag("userLimit", workspace);
 * if (userLimit === undefined) {
 *   return notFound();
 * }
 * ```
 */
export function flag<T extends keyof ConfigObject>(
  flagName: T,
  workspace: Partial<WorkspaceFeatures>,
): FlagValue<T> | null {
  if (process.env.NODE_ENV === "development") {
    return true as FlagValue<T>;
  }

  if (!workspace) {
    return null;
  }

  const betaFeature = workspace.betaFeatures?.[flagName as keyof Workspace["betaFeatures"]];

  if (betaFeature) {
    return betaFeature as FlagValue<T>;
  }

  const feature = workspace.features?.[flagName as keyof Workspace["features"]];

  if (feature) {
    return feature as FlagValue<T>;
  }

  return null;
}
