"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { useParams } from "next/navigation";
import { createContext, useContext } from "react";
import { useProjectData } from "../../data-provider";

type DeploymentLayoutContextType = {
  deployment: Deployment;
};

const DeploymentLayoutContext = createContext<DeploymentLayoutContextType | null>(null);

export const DeploymentLayoutProvider = ({ children }: { children: React.ReactNode }) => {
  const params = useParams();
  const deploymentId = params?.deploymentId;

  if (!deploymentId || typeof deploymentId !== "string") {
    throw new Error("DeploymentLayoutProvider must be used within a deployment route");
  }

  const { getDeploymentById, isDeploymentsLoading } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  if (isDeploymentsLoading) {
    return null;
  }

  if (!deployment) {
    throw new Error(`Deployment not found: ${deploymentId}`);
  }

  return (
    <DeploymentLayoutContext.Provider value={{ deployment }}>
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
