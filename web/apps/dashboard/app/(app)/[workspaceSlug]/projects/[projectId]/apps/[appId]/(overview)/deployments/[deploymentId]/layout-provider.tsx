"use client";

import { LoadingState } from "@/components/loading-state";
import { type Deployment, useDeployment as useDeploymentById } from "@/lib/deploy-data";
import { notFound, useParams } from "next/navigation";
import { createContext, useContext } from "react";

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

  // Loads the row with its instances however old it is; the scope's polling
  // keeps it live while the deployment builds
  const { data: resolved } = useDeploymentById(deploymentId);
  if (!resolved) {
    if (resolved === undefined) {
      return (
        <div className="flex min-h-(--app-body-h) flex-col">
          <LoadingState message="Loading deployment..." />
        </div>
      );
    }
    notFound();
  }
  return (
    <DeploymentLayoutContext.Provider value={{ deployment: resolved }}>
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
