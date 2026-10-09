"use client";

import { LoadError } from "@/components/load-error";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { productionFirst } from "@/lib/collections/deploy/environments";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, type ReactNode, createContext, use } from "react";
import { useAppId, useProjectData } from "../../[appId]/(overview)/data-provider";
import { SettingsSkeleton } from "./settings-skeleton";

type SettingsTransaction = ReturnType<typeof collection.environmentSettings.update>;

type EnvironmentSettingsContextType = {
  settings: EnvironmentSettings;
  update: (updater: (draft: EnvironmentSettings) => void) => SettingsTransaction;
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
  const { projectId } = useProjectData();
  const appId = useAppId();
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
  const { environments, isEnvironmentsLoading } = useProjectData();
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

  // Every environment has settings, because the defaults are written at create
  // time, so this is only empty while the request is in flight or after it fails.
  if (!settings) {
    return settingsLoad.failed ? <SettingsLoadError /> : fallback;
  }

  // One transaction for every target: the collection refetches after each
  // transaction settles, so one per environment would multiply the reads.
  const update = (updater: (draft: EnvironmentSettings) => void) =>
    collection.environmentSettings.update(targetIds, (drafts) => drafts.forEach(updater));

  return (
    <EnvironmentSettingsContext.Provider value={{ settings, update }}>
      {children}
    </EnvironmentSettingsContext.Provider>
  );
}

function SettingsLoadError() {
  const { retry } = useCollectionLoad(
    collection.environments.utils,
    collection.environmentSettings.utils,
  );
  return <LoadError title="Could not load settings" onRetry={retry} />;
}
