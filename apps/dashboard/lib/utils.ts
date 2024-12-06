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

type FlagValue<T extends keyof ConfigObject> = NonNullable<ConfigObject[T]>;

/**
 * Checks if a workspace has access to a specific feature or beta feature.
 * In development environment, returns devFallback value.
 * In production, returns the feature value if explicitly set, otherwise returns prodFallback.
 *
 * @param workspace - The workspace to check access for
 * @param flagName - The name of the feature to check
 * @param options - Configuration options
 * @param options.devFallback - Value to return in development environment
 * @param options.prodFallback - Value to return in production when feature is not set
 * @returns The feature value (boolean | number | string) based on environment and settings
 *
 * @example
 * ```typescript
 * // Check if workspace has access to logs page
 * if (!getFlag(workspace, "logsPage", {
 *   devFallback: true,   // Allow in development
 *   prodFallback: false  // Deny in production if not set
 * })) {
 *   return notFound();
 * }
 *
 * // Check feature with numeric value
 * const userLimit = getFlag(workspace, "userLimit", {
 *   devFallback: 1000,  // Higher limit in development
 *   prodFallback: 100   // Lower limit in production if not set
 * });
 *
 * // Check feature with string value
 * const tier = getFlag(workspace, "serviceTier", {
 *   devFallback: "premium",  // Use premium in development
 *   prodFallback: "basic"    // Use basic in production if not set
 * });
 * ```
 */
export function getFlag<TFlagName extends keyof ConfigObject>(
  workspace: Partial<WorkspaceFeatures>,
  flagName: TFlagName,
  {
    devFallback,
    prodFallback,
  }: {
    devFallback: FlagValue<TFlagName>;
    prodFallback: FlagValue<TFlagName>;
  },
): FlagValue<TFlagName> {
  if (process.env.NODE_ENV === "development") {
    return devFallback;
  }

  if (!workspace) {
    throw new Error(
      "Cannot get feature flag: No workspace found in database. Please verify workspace exists in the database or create a new workspace record.",
    );
  }

  const betaFeature = workspace.betaFeatures?.[flagName as keyof Workspace["betaFeatures"]];
  if (betaFeature !== undefined) {
    return betaFeature as FlagValue<TFlagName>;
  }

  const feature = workspace.features?.[flagName as keyof Workspace["features"]];
  if (feature !== undefined) {
    return feature as FlagValue<TFlagName>;
  }

  return prodFallback;
}
