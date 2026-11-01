"use client";
import { DEPLOYMENT_PICKER_LIMIT, useEnvironments, useRecentDeployments } from "@/lib/deploy-data";

export const useDiffDeployments = () => {
  const recent = useRecentDeployments({ limit: DEPLOYMENT_PICKER_LIMIT });
  const environments = useEnvironments();
  const deployments = recent.data.flatMap((deployment) => {
    const environment = environments.data.find((e) => e.id === deployment.environmentId);
    return environment ? [{ deployment, environment }] : [];
  });
  return { deployments, isLoading: recent.isLoading || environments.isLoading };
};
