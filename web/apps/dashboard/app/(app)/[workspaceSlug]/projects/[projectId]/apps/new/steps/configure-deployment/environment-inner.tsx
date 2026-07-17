"use client";

import { EnvironmentContext } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/settings/environment-provider";
import { collection } from "@/lib/collections";
import { useSettingsIsSaving } from "@/lib/collections/deploy/environment-settings";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, useEffect, useMemo } from "react";

export const OnboardingEnvironmentSettingsInner = ({
  children,
  prodEnvId,
  environments,
  onSettingsReady,
}: PropsWithChildren<{
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
        .where(({ s }) => eq(s.environmentId, prodEnvId)),
    [prodEnvId],
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
