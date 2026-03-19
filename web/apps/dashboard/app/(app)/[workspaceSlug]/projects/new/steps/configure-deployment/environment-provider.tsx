"use client";

import { type PropsWithChildren, useMemo } from "react";
import { useProjectData } from "../../../[projectId]/(overview)/data-provider";
import { OnboardingEnvironmentSettingsInner } from "./environment-inner";

/**
 * Drop-in replacement for EnvironmentSettingsProvider used during onboarding.
 *
 * Provides the same EnvironmentContext (so useEnvironmentSettings() works).
 * Returns null until environments have loaded so prodEnvId is always defined
 * before any live queries run.
 */
export const OnboardingEnvironmentSettingsProvider = ({ children }: PropsWithChildren) => {
  const { environments, isEnvironmentsLoading } = useProjectData();

  const prodEnvId = useMemo(
    () => (environments.find((e) => e.slug === "production") ?? environments.at(0))?.id,
    [environments],
  );

  // This is actually guarded by fallback component at where we call this provider
  if (isEnvironmentsLoading || !prodEnvId) {
    return null;
  }

  return (
    <OnboardingEnvironmentSettingsInner prodEnvId={prodEnvId} environments={environments}>
      {children}
    </OnboardingEnvironmentSettingsInner>
  );
};
