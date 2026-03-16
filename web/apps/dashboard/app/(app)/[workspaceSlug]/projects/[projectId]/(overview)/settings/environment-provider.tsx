"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useSearchParams } from "next/navigation";
import { type PropsWithChildren, createContext, useCallback, useContext } from "react";
import { useProjectData } from "../data-provider";

type EnvironmentContextType = {
  settings: EnvironmentSettings;
  variant: "settings" | "onboarding";
};

export const EnvironmentContext = createContext<EnvironmentContextType | null>(null);

export const EnvironmentSettingsProvider = ({ children }: PropsWithChildren) => {
  const { environments } = useProjectData();
  const searchParams = useSearchParams();
  const envIdParam = searchParams.get("environmentId");

  const getActiveEnvId = useCallback(() => {
    if (envIdParam) {
      const match = environments.find((e) => e.id === envIdParam);
      if (match) {
        return match.id;
      }
    }
    return environments.find((e) => e.slug === "production")?.id ?? environments.at(0)?.id;
  }, [envIdParam, environments]);

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, getActiveEnvId())),
    [getActiveEnvId],
  );

  console.log(data)

  // Selected envs settings cannot but null because we apply some sane defaults to them
  const settings = data.at(0);
  if (!settings) {
    return null;
  }

  return (
    <EnvironmentContext.Provider value={{ settings, variant: "settings" }}>
      {children}
    </EnvironmentContext.Provider>
  );
};

export function useEnvironmentSettings(): EnvironmentContextType {
  const context = useContext(EnvironmentContext);
  if (!context) {
    throw new Error("useEnvironmentSettings must be used within EnvironmentProvider");
  }
  return context;
}
