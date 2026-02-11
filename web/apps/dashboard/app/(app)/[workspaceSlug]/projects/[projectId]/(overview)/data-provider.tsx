"use client";

import { collection } from "@/lib/collections";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { createContext, useContext, useMemo } from "react";

type ProjectDataContextType = {
  // Cached project-level data
  domains: Domain[];
  deployments: Deployment[];

  isDomainsLoading: boolean;
  isDeploymentsLoading: boolean;

  getDomainsForDeployment: (deploymentId: string) => Domain[];
  getLiveDomains: () => Domain[];
  getEnvironmentOrLiveDomains: () => Domain[];
  getDeploymentById: (id: string) => Deployment | undefined;

  refetchDomains: () => void;
  refetchDeployments: () => void;
  refetchAll: () => void;
};

const ProjectDataContext = createContext<ProjectDataContextType | null>(null);

export const ProjectDataProvider = ({
  children,
  projectId,
}: {
  children: React.ReactNode;
  projectId: string;
}) => {
  const domainsQuery = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) => eq(domain.projectId, projectId))
        .orderBy(({ domain }) => domain.createdAt, "desc"),
    [projectId],
  );

  const deploymentsQuery = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) => eq(deployment.projectId, projectId))
        .orderBy(({ deployment }) => deployment.createdAt, "desc"),
    [projectId],
  );

  const value = useMemo(() => {
    const domains = domainsQuery.data ?? [];
    const deployments = deploymentsQuery.data ?? [];

    return {
      domains,
      deployments,
      isDomainsLoading: domainsQuery.isLoading,
      isDeploymentsLoading: deploymentsQuery.isLoading,

      getDomainsForDeployment: (deploymentId: string) =>
        domains.filter((d) => d.deploymentId === deploymentId),

      getLiveDomains: () => domains.filter((d) => d.sticky === "live"),

      getEnvironmentOrLiveDomains: () =>
        domains.filter((d) => d.sticky === "environment" || d.sticky === "live"),

      getDeploymentById: (id: string) => deployments.find((d) => d.id === id),

      refetchDomains: () => collection.domains.utils.refetch(),
      refetchDeployments: () => collection.deployments.utils.refetch(),
      refetchAll: () => {
        collection.projects.utils.refetch();
        collection.deployments.utils.refetch();
        collection.domains.utils.refetch();
      },
    };
  }, [domainsQuery, deploymentsQuery]);

  return <ProjectDataContext.Provider value={value}>{children}</ProjectDataContext.Provider>;
};

export const useProjectData = () => {
  const context = useContext(ProjectDataContext);
  if (!context) {
    throw new Error("useProjectData must be used within ProjectDataProvider");
  }
  return context;
};
