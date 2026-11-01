"use client";

import { LoadError } from "@/components/load-error";
import {
  type EnvironmentSettings,
  productionFirst,
  useAppEnvironmentSettings,
  useEnvironmentSettingsMutations,
  useEnvironments,
} from "@/lib/deploy-data";
import { type PropsWithChildren, type ReactNode, createContext, use } from "react";
import { SettingsSkeleton } from "./settings-skeleton";

type EnvironmentSettingsContextType = {
  settings: EnvironmentSettings;
  // Resolves when every environment of the scope has the change
  update: (
    updater: (draft: EnvironmentSettings) => void,
    config?: { optimistic: boolean },
  ) => Promise<unknown>;
};

type ScopeProps = PropsWithChildren<{ fallback?: ReactNode }>;

const EnvironmentSettingsContext = createContext<EnvironmentSettingsContextType | null>(null);

export function useEnvironmentSettings(): EnvironmentSettingsContextType {
  const context = use(EnvironmentSettingsContext);
  if (!context) {
    throw new Error("useEnvironmentSettings must be used within EnvironmentSettingsProvider");
  }
  return context;
}

export function EnvironmentSettingsProvider({
  children,
  fallback = <SettingsSkeleton />,
}: ScopeProps) {
  const { data: environments, isLoading: isEnvironmentsLoading } = useEnvironments();
  const { data: all } = useAppEnvironmentSettings();
  const shown = productionFirst(environments).at(0);

  if (isEnvironmentsLoading) {
    return fallback;
  }
  if (!shown) {
    return <SettingsLoadError />;
  }

  return (
    <SettingsContext
      settings={all.find((s) => s.environmentId === shown.id)}
      targetIds={environments.map((e) => e.id)}
      fallback={fallback}
    >
      {children}
    </SettingsContext>
  );
}

export function EnvironmentSettingsScope({
  children,
  environmentId,
  fallback = <SettingsSkeleton />,
}: ScopeProps & { environmentId: string }) {
  const { data: all } = useAppEnvironmentSettings();
  return (
    <SettingsContext
      settings={all.find((s) => s.environmentId === environmentId)}
      targetIds={[environmentId]}
      fallback={fallback}
    >
      {children}
    </SettingsContext>
  );
}

function SettingsContext({
  settings,
  targetIds,
  fallback,
  children,
}: ScopeProps & { settings: EnvironmentSettings | undefined; targetIds: string[] }) {
  const settingsLoad = useAppEnvironmentSettings();
  const { updateEnvironmentSettings } = useEnvironmentSettingsMutations();

  if (!settings) {
    return settingsLoad.isError ? <SettingsLoadError /> : fallback;
  }

  const update: EnvironmentSettingsContextType["update"] = (updater, config) =>
    updateEnvironmentSettings(targetIds, updater, config);

  return (
    <EnvironmentSettingsContext.Provider value={{ settings, update }}>
      {children}
    </EnvironmentSettingsContext.Provider>
  );
}

function SettingsLoadError() {
  const environments = useEnvironments();
  const settingsLoad = useAppEnvironmentSettings();
  return (
    <LoadError
      title="Could not load settings"
      onRetry={() => {
        environments.refetch();
        settingsLoad.refetch();
      }}
    />
  );
}
