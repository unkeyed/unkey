"use client";

import { collection } from "@/lib/collections";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Project } from "@/lib/collections/deploy/projects";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
import { type PropsWithChildren, createContext, useContext, useMemo } from "react";

type ProjectDataContextType = {
  projectId: string;

  project: Project | undefined;
  isProjectLoading: boolean;

  domains: Domain[];
  deployments: Deployment[];
  environments: Environment[];
  customDomains: CustomDomain[];

  isDomainsLoading: boolean;
  isDeploymentsLoading: boolean;
  isEnvironmentsLoading: boolean;
  isCustomDomainsLoading: boolean;

  getDomainsForDeployment: (deploymentId: string) => Domain[];
  getLiveDomains: () => Domain[];
  getEnvironmentOrLiveDomains: () => Domain[];
  getDeploymentById: (id: string) => Deployment | undefined;

  refetchDomains: () => void;
  refetchDeployments: () => void;
  refetchCustomDomains: () => void;
  refetchAll: () => void;
};

const ProjectDataContext = createContext<ProjectDataContextType | null>(null);

export const ProjectDataProvider = ({ children }: PropsWithChildren) => {
  const params = useParams();
  const projectId = params?.projectId;

  if (!projectId || typeof projectId !== "string") {
    throw new Error("ProjectDataProvider must be used within a project route");
  }

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

  const projectQuery = useLiveQuery(
    (q) =>
      q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
    [projectId],
  );

  const environmentsQuery = useLiveQuery(
    (q) =>
      q.from({ env: collection.environments }).where(({ env }) => eq(env.projectId, projectId)),
    [projectId],
  );

  const customDomainsQuery = useLiveQuery(
    (q) =>
      q
        .from({ customDomain: collection.customDomains })
        .where(({ customDomain }) => eq(customDomain.projectId, projectId))
        .orderBy(({ customDomain }) => customDomain.createdAt, "desc"),
    [projectId],
  );

  const value = useMemo(() => {
    const domains = domainsQuery.data ?? [];
    const deployments = deploymentsQuery.data ?? [];
    const environments = environmentsQuery.data ?? [];
    const customDomains = customDomainsQuery.data ?? [];
    const project = projectQuery.data?.at(0);

    return {
      projectId,

      project,
      isProjectLoading: projectQuery.isLoading,

      domains,
      isDomainsLoading: domainsQuery.isLoading,

      deployments,
      isDeploymentsLoading: deploymentsQuery.isLoading,

      environments,
      isEnvironmentsLoading: environmentsQuery.isLoading,

      customDomains,
      isCustomDomainsLoading: customDomainsQuery.isLoading,

      getDomainsForDeployment: (deploymentId: string) =>
        domains.filter((d) => d.deploymentId === deploymentId),

      getLiveDomains: () => domains.filter((d) => d.sticky === "live"),

      getEnvironmentOrLiveDomains: () =>
        domains.filter((d) => d.sticky === "environment" || d.sticky === "live"),

      getDeploymentById: (id: string) => deployments.find((d) => d.id === id),

      refetchDomains: () => collection.domains.utils.refetch(),
      refetchDeployments: () => collection.deployments.utils.refetch(),
      refetchCustomDomains: () => collection.customDomains.utils.refetch(),
      refetchAll: () => {
        collection.projects.utils.refetch();
        collection.deployments.utils.refetch();
        collection.domains.utils.refetch();
        collection.environments.utils.refetch();
        collection.customDomains.utils.refetch();
      },
    };
  }, [projectId, domainsQuery, deploymentsQuery, projectQuery, environmentsQuery, customDomainsQuery]);

  return <ProjectDataContext.Provider value={value}>{children}</ProjectDataContext.Provider>;
};

export const useProjectData = () => {
  const context = useContext(ProjectDataContext);
  if (!context) {
    throw new Error("useProjectData must be used within ProjectDataProvider");
  }
  return context;
};
