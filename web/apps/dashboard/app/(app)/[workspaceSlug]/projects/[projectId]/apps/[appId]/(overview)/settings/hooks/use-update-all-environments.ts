"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { useCallback } from "react";
import { useProjectData } from "../../data-provider";

type SettingsTransaction = ReturnType<typeof collection.environmentSettings.update>;

/**
 * Returns a function that applies a settings mutation to every environment.
 *
 * Use this for settings that don't have per-environment UI (e.g. dockerfile,
 * root directory, port, command, healthcheck) so they stay consistent across
 * all environments.
 */
export function useUpdateAllEnvironments() {
  const { environments } = useProjectData();

  return useCallback(
    (updater: (draft: EnvironmentSettings) => void): SettingsTransaction | null => {
      if (environments.length === 0) {
        return null;
      }
      // One transaction for every environment. The collection refetches each
      // loaded environment after a transaction settles, so a transaction per
      // environment would multiply the reads.
      return collection.environmentSettings.update(
        environments.map((env) => env.id),
        (drafts) => drafts.forEach(updater),
      );
    },
    [environments],
  );
}
