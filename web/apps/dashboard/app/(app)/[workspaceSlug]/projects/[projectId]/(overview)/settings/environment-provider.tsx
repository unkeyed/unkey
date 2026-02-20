"use client";

import { type PropsWithChildren, createContext, useContext, useMemo } from "react";
import { useProjectData } from "../data-provider";

type EnvironmentContextType = {
  environmentId: string;
};

const EnvironmentContext = createContext<EnvironmentContextType | null>(null);

export const EnvironmentProvider = ({ children }: PropsWithChildren) => {
  const { environments } = useProjectData();

  // Later we'll switch this useProjectData with useParams and directly consume environmentId from there.
  // This provider will make that transition easier
  const value = useMemo(() => {
    // Temporary fallback 
    const env = environments.find((e) => e.slug === "production") ?? environments[0];
    if (!env) {
      return null;
    }
    return { environmentId: env.id };
  }, [environments]);

  if (!value) {
    return null;
  }

  return <EnvironmentContext.Provider value={value}>{children}</EnvironmentContext.Provider>;
};

export const useEnvironmentId = (): string => {
  const context = useContext(EnvironmentContext);
  if (!context) {
    throw new Error("useEnvironmentId must be used within EnvironmentProvider");
  }
  return context.environmentId;
};
