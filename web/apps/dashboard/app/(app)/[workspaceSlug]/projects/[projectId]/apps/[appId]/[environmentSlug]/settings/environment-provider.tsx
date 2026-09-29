"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, createContext, useContext } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { useAppEnvironment } from "../environment-context";
import { SettingsSkeleton } from "./components/settings-skeleton";

type EnvironmentContextType = {
  settings: EnvironmentSettings;
  variant: "settings" | "onboarding";
  isSaving: boolean;
};

export const EnvironmentContext = createContext<EnvironmentContextType | null>(null);

export const EnvironmentSettingsProvider = ({ children }: PropsWithChildren) => {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const { environment } = useAppEnvironment();

  return (
    <EnvironmentSettingsInner projectId={projectId} appId={appId} environmentId={environment.id}>
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
