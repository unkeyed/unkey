"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { useCallback } from "react";
import { useEnvironmentSettings } from "../environment-provider";
import { useUpdateAllEnvironments } from "./use-update-all-environments";

/**
 * Applies a per-environment setting (cpu, regions, auto deploy, ...) to the
 * environment in scope. Onboarding has no environment scope yet, so its one
 * form seeds every environment.
 */
export function useUpdateEnvironment() {
  const { settings, variant } = useEnvironmentSettings();
  const updateAllEnvironments = useUpdateAllEnvironments();

  return useCallback(
    (updater: (draft: EnvironmentSettings) => void) => {
      if (variant === "onboarding") {
        updateAllEnvironments(updater);
        return;
      }
      collection.environmentSettings.update(settings.environmentId, updater);
    },
    [variant, settings.environmentId, updateAllEnvironments],
  );
}
