"use client";

import { LoadingState } from "@/components/loading-state";
import { type Deployment, deploymentSchema } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import { notFound, useParams } from "next/navigation";
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

  const { getDeploymentById, isDeploymentsLoading, projectId } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  const { data: fetchedDeployment, isLoading: isFetchingById } =
    trpc.deploy.deployment.getById.useQuery({ deploymentId, projectId }, { enabled: !deployment });

  const parsed = fetchedDeployment ? deploymentSchema.safeParse(fetchedDeployment) : undefined;
  const resolved = deployment ?? (parsed?.success ? parsed.data : undefined);
  if (!resolved) {
    if (isDeploymentsLoading || isFetchingById) {
      return <LoadingState message="Loading deployment..." />;
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
