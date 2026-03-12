"use client";

import { collection } from "@/lib/collections";
import {
  subscribeToSettingsSaving,
} from "@/lib/collections/deploy/environment-settings";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { type PropsWithChildren, useEffect, useMemo, useState } from "react";
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

  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    return subscribeToSettingsSaving(setIsSaving);
  }, []);

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
