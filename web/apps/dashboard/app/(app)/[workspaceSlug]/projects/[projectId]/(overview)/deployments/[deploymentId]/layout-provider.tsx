"use client";

import { useParams } from "next/navigation";
import { createContext, useContext } from "react";

type DeploymentLayoutContextType = {
  deploymentId: string;
};

const DeploymentLayoutContext = createContext<DeploymentLayoutContextType | null>(null);

export const DeploymentLayoutProvider = ({ children }: { children: React.ReactNode }) => {
  const params = useParams();
  const deploymentId = params?.deploymentId;

  if (!deploymentId || typeof deploymentId !== "string") {
    throw new Error("DeploymentLayoutProvider must be used within a deployment route");
  }

  return (
    <DeploymentLayoutContext.Provider value={{ deploymentId }}>
      {children}
    </DeploymentLayoutContext.Provider>
  );
};

export const useDeployment = () => {
  const context = useContext(DeploymentLayoutContext);
  if (!context) {
    throw new Error("useDeployment must be used within a deployment route");
  }
  return context;
};
