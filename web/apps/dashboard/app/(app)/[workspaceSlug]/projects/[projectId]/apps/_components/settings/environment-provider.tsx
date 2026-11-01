"use client";

import { LoadError } from "@/components/load-error";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { productionFirst, useAppScope, useEnvironments } from "@/lib/deploy-data";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, type ReactNode, createContext, use } from "react";
import { SettingsSkeleton } from "./settings-skeleton";

type SettingsTransaction = ReturnType<typeof collection.environmentSettings.update>;

type EnvironmentSettingsContextType = {
  settings: EnvironmentSettings;
  update: (
    updater: (draft: EnvironmentSettings) => void,
    config?: { optimistic: boolean },
  ) => SettingsTransaction;
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

export function useAppEnvironmentSettings(): EnvironmentSettings[] {
  const { projectId, appId } = useAppScope();
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => and(eq(s.projectId, projectId), eq(s.appId, appId))),
    [projectId, appId],
  );
  return data;
}

export function EnvironmentSettingsProvider({
  children,
  fallback = <SettingsSkeleton />,
}: ScopeProps) {
  const { data: environments, isLoading: isEnvironmentsLoading } = useEnvironments();
  const all = useAppEnvironmentSettings();
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
  const all = useAppEnvironmentSettings();
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
  const settingsLoad = useCollectionLoad(collection.environmentSettings.utils);

  if (!settings) {
    return settingsLoad.failed ? <SettingsLoadError /> : fallback;
  }

  // One transaction for every target: the collection refetches after each
  // transaction settles, so one per environment would multiply the reads.
  // Saves are not optimistic by default, so a failed save leaves the row and the form's edits alone.
  const update: EnvironmentSettingsContextType["update"] = (updater, config) =>
    collection.environmentSettings.update(
      targetIds,
      { optimistic: config?.optimistic ?? false },
      (drafts) => drafts.forEach(updater),
    );

  return (
    <EnvironmentSettingsContext.Provider value={{ settings, update }}>
      {children}
    </EnvironmentSettingsContext.Provider>
  );
}

function SettingsLoadError() {
  const environments = useEnvironments();
  const settingsLoad = useCollectionLoad(collection.environmentSettings.utils);
  return (
    <LoadError
      title="Could not load settings"
      onRetry={() => {
        environments.refetch();
        settingsLoad.retry();
      }}
    />
  );
}
