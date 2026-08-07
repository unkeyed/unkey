"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useSearchParams } from "next/navigation";
import { type PropsWithChildren, createContext, useContext, useMemo } from "react";
import { useProjectData } from "../data-provider";
import { SettingsSkeleton } from "./components/settings-skeleton";

type EnvironmentContextType = {
  settings: EnvironmentSettings;
  variant: "settings" | "onboarding";
  isSaving: boolean;
};

export const EnvironmentContext = createContext<EnvironmentContextType | null>(null);

/**
 * Resolves the environment to show, then hands off to the inner provider.
 *
 * The settings query needs a project, an app, and an environment, and the query
 * builder rejects an undefined value. Waiting here keeps the inner query free of
 * placeholder ids.
 */
export const EnvironmentSettingsProvider = ({ children }: PropsWithChildren) => {
  const { environments, isEnvironmentsLoading, projectId, appId } = useProjectData();
  const searchParams = useSearchParams();
  const envIdParam = searchParams.get("environmentId");

  const activeEnvironmentId = useMemo(() => {
    if (envIdParam) {
      const match = environments.find((e) => e.id === envIdParam);
      if (match) {
        return match.id;
      }
    }
    return environments.find((e) => e.kind === "production")?.id ?? environments.at(0)?.id;
  }, [envIdParam, environments]);

  if (isEnvironmentsLoading || !activeEnvironmentId || !appId) {
    return <SettingsSkeleton />;
  }

  return (
    <EnvironmentSettingsInner
      projectId={projectId}
      appId={appId}
      environmentId={activeEnvironmentId}
    >
      {children}
    </EnvironmentSettingsInner>
  );
};

export function useEnvironmentSettings(): EnvironmentContextType {
  const context = useContext(EnvironmentContext);
  if (!context) {
    throw new Error("useEnvironmentSettings must be used within EnvironmentProvider");
  }
  return context;
}

const EnvironmentSettingsInner = ({
  children,
  projectId,
  appId,
  environmentId,
}: PropsWithChildren<{ projectId: string; appId: string; environmentId: string }>) => {
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) =>
          and(eq(s.projectId, projectId), eq(s.appId, appId), eq(s.environmentId, environmentId)),
        ),
    [projectId, appId, environmentId],
  );

  // Every environment has settings, because the defaults are written at create
  // time, so this is only empty while the request is in flight.
  const settings = data.at(0);
  if (!settings) {
    return <SettingsSkeleton />;
  }

  return (
    <EnvironmentContext.Provider value={{ settings, variant: "settings", isSaving: false }}>
      {children}
    </EnvironmentContext.Provider>
  );
};
