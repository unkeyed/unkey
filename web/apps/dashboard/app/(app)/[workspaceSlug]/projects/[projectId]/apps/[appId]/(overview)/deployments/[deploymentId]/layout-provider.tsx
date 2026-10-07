"use client";

import { LoadingState } from "@/components/loading-state";
import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { useLiveQuery } from "@tanstack/react-db";
import { notFound, useParams } from "next/navigation";
import { createContext, useContext } from "react";
import { useProjectData } from "../../data-provider";
import { deploymentQueryFor } from "../../data-provider-queries";

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

  const { projectId } = useProjectData();
  // A single-id query loads the row with its instances however old it is, and
  // the provider's refetch keeps it live while the deployment builds
  const deploymentQuery = useLiveQuery(deploymentQueryFor(projectId, deploymentId), [
    projectId,
    deploymentId,
  ]);

  const resolved = deploymentQuery.data?.[0];
  if (!resolved) {
    if (deploymentQuery.isLoading) {
      return (
        <div className="flex flex-col" style={{ height: `calc(100dvh - ${TOP_NAV_HEIGHT}px)` }}>
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
