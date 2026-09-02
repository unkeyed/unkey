"use client";

import { useCallback } from "react";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
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
      if (environments.length === 0) {
        return;
      }
      // One transaction for every environment. The collection refetches each
      // loaded environment after a transaction settles, so a transaction per
      // environment would multiply the reads.
      collection.environmentSettings.update(
        environments.map((env) => env.id),
        (drafts) => drafts.forEach(updater),
      );
    },
    [environments],
  );
}
