"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, type ReactNode, createContext, use } from "react";
import { LoadError } from "../../components/load-error";
import { useAppId, useProjectData } from "../data-provider";
import { SettingsSkeleton } from "./components/settings-skeleton";

type EnvironmentContextType = {
  settings: EnvironmentSettings;
  autoSave: boolean;
};

type ScopeProps = PropsWithChildren<{ autoSave?: boolean; fallback?: ReactNode }>;

const EnvironmentContext = createContext<EnvironmentContextType | null>(null);

/**
 * Scopes the page to the production environment's settings.
 *
 * The settings query needs a project, an app, and an environment, and the query
 * builder rejects an undefined value. Waiting here keeps the scoped query free
 * of placeholder ids.
 */
export function EnvironmentSettingsProvider({
  children,
  autoSave = false,
  fallback = <SettingsSkeleton />,
}: ScopeProps) {
  const { environments, isEnvironmentsLoading, projectId } = useProjectData();
  const appId = useAppId();
  // Loads the settings alongside the environments, so both share one listEnvironments request.
  useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => and(eq(s.projectId, projectId), eq(s.appId, appId))),
    [projectId, appId],
  );
  const activeEnvironmentId =
    environments.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id ?? environments.at(0)?.id;

  if (isEnvironmentsLoading) {
    return fallback;
  }
  if (!activeEnvironmentId) {
    return <SettingsLoadError />;
  }

  return (
    <EnvironmentSettingsScope
      environmentId={activeEnvironmentId}
      autoSave={autoSave}
      fallback={fallback}
    >
      {children}
    </EnvironmentSettingsScope>
  );
}

export function useEnvironmentSettings(): EnvironmentContextType {
  const context = use(EnvironmentContext);
  if (!context) {
    throw new Error("useEnvironmentSettings must be used within EnvironmentProvider");
  }
  return context;
}

export function EnvironmentSettingsScope({
  children,
  environmentId,
  autoSave = false,
  fallback = <SettingsSkeleton />,
}: ScopeProps & { environmentId: string }) {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) =>
          and(eq(s.projectId, projectId), eq(s.appId, appId), eq(s.environmentId, environmentId)),
        ),
    [projectId, appId, environmentId],
  );

  const settingsLoad = useCollectionLoad(collection.environmentSettings.utils);

  // Every environment has settings, because the defaults are written at create
  // time, so this is only empty while the request is in flight or after it fails.
  const settings = data.at(0);
  if (!settings) {
    return settingsLoad.failed ? <SettingsLoadError /> : fallback;
  }

  return (
    <EnvironmentContext.Provider value={{ settings, autoSave }}>
      {children}
    </EnvironmentContext.Provider>
  );
}

function SettingsLoadError() {
  const { retry } = useCollectionLoad(
    collection.environments.utils,
    collection.environmentSettings.utils,
  );
  return <LoadError title="Could not load settings" onRetry={retry} />;
}
