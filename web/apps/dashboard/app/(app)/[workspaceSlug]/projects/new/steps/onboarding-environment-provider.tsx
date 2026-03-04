"use client";
import { collection } from "@/lib/collections";
import {
  type EnvironmentSettings,
  buildSettingsMutations,
} from "@/lib/collections/deploy/environment-settings";
import { trpc } from "@/lib/trpc/client";
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
export const OnboardingEnvironmentSettingsProvider = ({
  children,
  isActive,
}: PropsWithChildren<{ isActive: boolean }>) => {
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

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: Boolean(prodEnvId) },
  );

  useInitializeSettings(settings, availableRegions, isActive);
  useSyncSettingsToOtherEnvironments(settings, otherEnvIds);

  if (!settings) {
    return null;
  }

  return (
    <EnvironmentContext.Provider value={{ settings, autoSave: true }}>
      {children}
    </EnvironmentContext.Provider>
  );
};

// Settings are empty initially so we set all of them by default for the user.
// Later they can change it in the settings.
function useInitializeSettings(
  settings: EnvironmentSettings | undefined,
  availableRegions: string[] | undefined,
  isActive: boolean,
) {
  const hasInitializedRef = useRef(false);

  useEffect(() => {
    if (!settings || !availableRegions || !isActive) {
      return;
    }
    if (hasInitializedRef.current) {
      return;
    }
    hasInitializedRef.current = true;

    collection.environmentSettings.update(
      settings.environmentId,
      { metadata: { silent: true } },
      (draft) => {
        if (!draft.dockerfile) {
          draft.dockerfile = "Dockerfile";
        }
        if (!draft.dockerContext) {
          draft.dockerContext = ".";
        }
        if (!draft.port) {
          draft.port = 8080;
        }
        if (!draft.cpuMillicores) {
          draft.cpuMillicores = 256;
        }
        if (!draft.memoryMib) {
          draft.memoryMib = 256;
        }
        if (!draft.shutdownSignal) {
          draft.shutdownSignal = "SIGTERM";
        }
        if (Object.keys(draft.regionConfig).length === 0) {
          for (const region of availableRegions) {
            draft.regionConfig[region] = 1;
          }
        }
      },
    );
  }, [settings, availableRegions, isActive]);
}

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
