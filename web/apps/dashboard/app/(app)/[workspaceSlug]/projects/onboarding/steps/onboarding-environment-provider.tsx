"use client";
import { collection } from "@/lib/collections";
import {
  type EnvironmentSettings,
  buildSettingsMutations,
} from "@/lib/collections/deploy/environment-settings";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, useEffect, useMemo, useRef } from "react";
import { useProjectData } from "../../[projectId]/(overview)/data-provider";
import { EnvironmentContext } from "../../[projectId]/(overview)/settings/environment-provider";

/**
 * Drop-in replacement for EnvironmentSettingsProvider used during onboarding.
 *
 * Provides the same EnvironmentContext (so useEnvironmentSettings() works),
 * but syncs production setting changes to every other environment so new
 * projects start with consistent config.
 */
export const OnboardingEnvironmentSettingsProvider = ({ children }: PropsWithChildren) => {
  const { environments } = useProjectData();

  const prodEnvId = useMemo(
    () => (environments.find((e) => e.slug === "production") ?? environments.at(0))?.id,
    [environments],
  );

  const otherEnvIds = useMemo(
    () => environments.filter((e) => e.id !== prodEnvId).map((e) => e.id),
    [environments, prodEnvId],
  );

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, prodEnvId)),
    [prodEnvId],
  );

  const settings = data.at(0);

  useSyncSettingsToOtherEnvironments(settings, otherEnvIds);

  if (!settings) {
    return null;
  }

  return <EnvironmentContext.Provider value={{ settings }}>{children}</EnvironmentContext.Provider>;
};

function useSyncSettingsToOtherEnvironments(
  settings: EnvironmentSettings | undefined,
  otherEnvIds: string[],
) {
  const settingsRef = useRef(settings);
  settingsRef.current = settings;
  const otherEnvIdsRef = useRef(otherEnvIds);
  otherEnvIdsRef.current = otherEnvIds;
  const prevSettingsRef = useRef<EnvironmentSettings | null>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: JSON.stringify used for deep comparison; values read from refs to avoid stale closures
  useEffect(() => {
    const current = settingsRef.current;
    const envIds = otherEnvIdsRef.current;

    if (!current || envIds.length === 0) {
      return;
    }

    const prev = prevSettingsRef.current;
    prevSettingsRef.current = current;

    if (!prev) {
      return; // skip initial load
    }

    const mutations = envIds.flatMap((envId) => buildSettingsMutations(envId, prev, current));

    if (mutations.length > 0) {
      Promise.all(mutations).catch(console.error);
    }
  }, [JSON.stringify(settings), JSON.stringify(otherEnvIds)]);
}
