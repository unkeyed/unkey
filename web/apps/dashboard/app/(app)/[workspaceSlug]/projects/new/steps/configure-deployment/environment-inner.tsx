"use client";

import { collection } from "@/lib/collections";
import {
  ENVIRONMENT_SETTINGS_DEFAULTS,
  type EnvironmentSettings,
  buildSettingsMutations,
  useSettingsIsSaving,
} from "@/lib/collections/deploy/environment-settings";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { type PropsWithChildren, useEffect, useMemo, useRef } from "react";
import { EnvironmentContext } from "../../../[projectId]/(overview)/settings/environment-provider";

export const OnboardingEnvironmentSettingsInner = ({
  children,
  prodEnvId,
  environments,
}: PropsWithChildren<{
  prodEnvId: string;
  environments: { id: string; slug: string }[];
}>) => {
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

  const isSaving = useSettingsIsSaving();

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery(
    undefined,
    { enabled: Boolean(prodEnvId) },
  );

  useInitializeSettings(environments, availableRegions);

  // Setting cannot be null at this point coz they are preloaded
  if (!settings) {
    return null;
  }

  return (
    <EnvironmentContext.Provider value={{ settings, variant: "onboarding", isSaving }}>
      {otherEnvIds.map((id) => (
        <EnvironmentSettingsPreloader key={id} envId={id} />
      ))}
      {children}
    </EnvironmentContext.Provider>
  );
};

const EnvironmentSettingsPreloader = ({ envId }: { envId: string }) => {
  useLiveQuery(
    (q) =>
      q.from({ s: collection.environmentSettings }).where(({ s }) => eq(s.environmentId, envId)),
    [envId],
  );
  return null;
};

// Settings are empty initially so we persist defaults for every environment.
// Uses buildSettingsMutations directly to bypass the collection's onUpdate
// handler (which would show toasts and whose silent metadata flag is broken).
function useInitializeSettings(
  environments: { id: string; slug: string }[],
  availableRegions: { id: string; name: string }[] | undefined,
) {
  const hasInitializedRef = useRef(false);

  useEffect(() => {
    if (!availableRegions || environments.length === 0) {
      return;
    }
    if (hasInitializedRef.current) {
      return;
    }
    hasInitializedRef.current = true;

    const d = ENVIRONMENT_SETTINGS_DEFAULTS;
    const defaults = {
      dockerfile: d.dockerfile,
      dockerContext: d.dockerContext,
      watchPaths: [] as string[],
      port: d.port,
      cpuMillicores: d.cpuMillicores,
      memoryMib: d.memoryMib,
      command: [] as string[],
      healthcheck: null,
      regions: availableRegions.map((r) => ({ id: r.id, name: r.name, replicas: 1 })),
      shutdownSignal: d.shutdownSignal,
      openapiSpecPath: null,
    };

    const empty: EnvironmentSettings = {
      environmentId: "",
      dockerfile: "",
      dockerContext: "",
      watchPaths: [],
      port: 0,
      cpuMillicores: 0,
      memoryMib: 0,
      command: [],
      healthcheck: null,
      regions: [],
      shutdownSignal: "",
      openapiSpecPath: null,
    };

    const mutations = environments.flatMap((env) =>
      buildSettingsMutations(env.id, empty, { ...defaults, environmentId: env.id }),
    );

    if (mutations.length > 0) {
      Promise.all(mutations).catch((err) => {
        toast.error("Failed to initialize settings", {
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        });
      });
    }
  }, [environments, availableRegions]);
}
