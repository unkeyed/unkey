"use client";

import { collection } from "@/lib/collections";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Project } from "@/lib/collections/deploy/projects";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
import { type PropsWithChildren, createContext, useContext, useEffect, useMemo } from "react";

type ProjectDataContextType = {
  projectId: string;

  project: Project | undefined;
  isProjectLoading: boolean;

  domains: Domain[];
  deployments: Deployment[];
  environments: Environment[];

  isDomainsLoading: boolean;
  isDeploymentsLoading: boolean;
  isEnvironmentsLoading: boolean;

  getDomainsForDeployment: (deploymentId: string) => Domain[];
  getLiveDomains: () => Domain[];
  getEnvironmentOrLiveDomains: () => Domain[];
  getDeploymentById: (id: string) => Deployment | undefined;

  refetchDomains: () => void;
  refetchDeployments: () => void;
  refetchAll: () => void;
};

const ProjectDataContext = createContext<ProjectDataContextType | null>(null);

export const ProjectDataProvider = ({ children }: PropsWithChildren) => {
  const params = useParams();
  const projectId = params?.projectId;

  if (!projectId || typeof projectId !== "string") {
    throw new Error("ProjectDataProvider must be used within a project route");
  }

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

  const project = projectQuery.data?.at(0);
  const domainsQuery = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) => eq(domain.projectId, projectId))
        .orderBy(({ domain }) => domain.createdAt, "desc"),
    [projectId],
  );
  // refetch domains when live deployment changes
  useEffect(() => {
    if (project?.liveDeploymentId) {
      collection.domains.utils.refetch();
    }
  }, [project?.liveDeploymentId]);

  const environmentsQuery = useLiveQuery(
    (q) =>
      q.from({ env: collection.environments }).where(({ env }) => eq(env.projectId, projectId)),
    [projectId],
  );

  const value = useMemo(() => {
    const domains = domainsQuery.data ?? [];
    const deployments = deploymentsQuery.data ?? [];
    const environments = environmentsQuery.data ?? [];
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
        collection.environments.utils.refetch();
      },
    };
  }, [projectId, domainsQuery, deploymentsQuery, projectQuery, environmentsQuery]);

  return <ProjectDataContext.Provider value={value}>{children}</ProjectDataContext.Provider>;
};

export const useProjectData = () => {
  const context = useContext(ProjectDataContext);
  if (!context) {
    throw new Error("useProjectData must be used within ProjectDataProvider");
  }
  return context;
};
