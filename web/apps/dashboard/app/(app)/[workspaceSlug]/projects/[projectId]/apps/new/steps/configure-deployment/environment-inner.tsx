"use client";

import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, useEffect, useMemo } from "react";
import { EnvironmentContext } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/environment-provider";
import { collection } from "@/lib/collections";
import { useSettingsIsSaving } from "@/lib/collections/deploy/environment-settings";

export const OnboardingEnvironmentSettingsInner = ({
  children,
  projectId,
  appId,
  prodEnvId,
  environments,
  onSettingsReady,
}: PropsWithChildren<{
  projectId: string;
  appId: string;
  prodEnvId: string;
  environments: { id: string; slug: string }[];
  onSettingsReady: () => void;
}>) => {
  const otherEnvIds = useMemo(
    () => environments.filter((e) => e.id !== prodEnvId).map((e) => e.id),
    [environments, prodEnvId],
  );

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) =>
          and(eq(s.projectId, projectId), eq(s.appId, appId), eq(s.environmentId, prodEnvId)),
        ),
    [prodEnvId, projectId, appId],
  );

  const settings = data.at(0);

  const isSaving = useSettingsIsSaving();

  useEffect(() => {
    if (settings) {
      onSettingsReady();
    }
  }, [settings, onSettingsReady]);

  if (!settings) {
    return null;
  }

  return (
    <EnvironmentContext.Provider value={{ settings, variant: "onboarding", isSaving }}>
      {otherEnvIds.map((id) => (
        <EnvironmentSettingsPreloader key={id} projectId={projectId} appId={appId} envId={id} />
      ))}
      {children}
    </EnvironmentContext.Provider>
  );
};

const EnvironmentSettingsPreloader = ({
  projectId,
  appId,
  envId,
}: {
  projectId: string;
  appId: string;
  envId: string;
}) => {
  useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) =>
          and(eq(s.projectId, projectId), eq(s.appId, appId), eq(s.environmentId, envId)),
        ),
    [envId, projectId, appId],
  );
  return null;
};
