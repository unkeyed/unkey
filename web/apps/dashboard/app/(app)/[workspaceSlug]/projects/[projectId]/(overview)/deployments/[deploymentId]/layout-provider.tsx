"use client";

import { LoadingState } from "@/components/loading-state";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { useParams } from "next/navigation";
import { createContext, useContext } from "react";
import { useProjectData } from "../../data-provider";

type DeploymentLayoutContextType = {
  deployment: Deployment;
};

const DeploymentLayoutContext = createContext<DeploymentLayoutContextType | null>(null);

type DeploymentLayoutProviderProps = {
  children: React.ReactNode;
  deploymentId?: string;
};

export const DeploymentLayoutProvider = ({
  children,
  deploymentId: deploymentIdProp,
}: DeploymentLayoutProviderProps) => {
  const params = useParams();
  const deploymentId =
    deploymentIdProp ??
    (typeof params?.deploymentId === "string" ? params.deploymentId : undefined);

  if (!deploymentId) {
    throw new Error("DeploymentLayoutProvider requires a deploymentId (via prop or route params)");
  }

  const { getDeploymentById, isDeploymentsLoading } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  if (!deployment) {
    if (isDeploymentsLoading) {
      return <LoadingState message="Loading deployment..." />;
    }
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
