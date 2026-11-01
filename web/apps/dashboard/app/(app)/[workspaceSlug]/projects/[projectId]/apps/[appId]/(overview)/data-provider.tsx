"use client";

import { useProject } from "@/hooks/use-project";
import {
  useApp,
  useAppScope,
  useCustomDomains,
  useDeployActions,
  useDeployScope,
  useDomains,
  useEnvironments,
} from "@/lib/deploy-data";

// Legacy pages only. Everything else reads through @/lib/deploy-data.
export const useProjectData = () => {
  const { projectId, appId } = useDeployScope();
  const { refetchDeployments } = useDeployActions();
  const app = useApp();
  const { project } = useProject();
  const domains = useDomains();
  const customDomains = useCustomDomains();
  const environments = useEnvironments();
  return {
    projectId,
    appId,
    project: project
      ? {
          name: project.name,
          repositoryFullName: app.data?.repositoryFullName ?? null,
          currentDeploymentId: app.data?.currentDeploymentId ?? null,
        }
      : undefined,
    domains: domains.data,
    isDomainsLoading: domains.isLoading,
    refetchDomains: domains.refetch,
    customDomains: customDomains.data,
    isCustomDomainsLoading: customDomains.isLoading,
    environments: environments.data,
    isEnvironmentsLoading: environments.isLoading,
    refetchDeployments,
  };
};

export const useAppId = (): string => useAppScope().appId;
