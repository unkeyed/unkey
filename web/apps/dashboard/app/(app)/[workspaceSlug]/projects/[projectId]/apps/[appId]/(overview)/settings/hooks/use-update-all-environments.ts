"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { useCallback } from "react";
import { useProjectData } from "../../data-provider";

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
    (updater: (draft: EnvironmentSettings) => void) => {
      for (const env of environments) {
        collection.environmentSettings.update(env.id, updater);
      }
    },
    [environments],
  );
}
