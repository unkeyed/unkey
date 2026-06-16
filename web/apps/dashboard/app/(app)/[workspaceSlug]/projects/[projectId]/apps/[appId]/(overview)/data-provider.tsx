"use client";

import { collection } from "@/lib/collections";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import { DEPLOYMENTS_DEFAULT_LIMIT, type Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Project } from "@/lib/collections/deploy/projects";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { notFound, useParams } from "next/navigation";
import {
  type PropsWithChildren,
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from "react";

type ProjectDataContextType = {
  projectId: string;
  // Transitional: optional until the detail view and onboarding wizard move
  // under apps/[appId]. Becomes required (no project-wide fallback) once every
  // mount supplies an app.
  appId: string | undefined;

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

type ProjectDataProviderProps = PropsWithChildren<{ projectId?: string; appId?: string }>;

export const ProjectDataProvider = ({
  children,
  projectId: projectIdProp,
  appId: appIdProp,
}: ProjectDataProviderProps) => {
  const params = useParams();
  const projectId =
    projectIdProp ?? (typeof params?.projectId === "string" ? params.projectId : undefined);
  const appId = appIdProp ?? (typeof params?.appId === "string" ? params.appId : undefined);

  if (!projectId) {
    throw new Error("ProjectDataProvider requires a projectId prop or a [projectId] route param");
  }

  const deploymentsQuery = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          appId
            ? and(eq(deployment.projectId, projectId), eq(deployment.appId, appId))
            : eq(deployment.projectId, projectId),
        )
        .orderBy(({ deployment }) => deployment.createdAt, "desc")
        .limit(DEPLOYMENTS_DEFAULT_LIMIT),
    [projectId, appId],
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
        .where(({ domain }) =>
          appId
            ? and(eq(domain.projectId, projectId), eq(domain.appId, appId))
            : eq(domain.projectId, projectId),
        )
        .orderBy(({ domain }) => domain.createdAt, "desc"),
    [projectId, appId],
  );
  // refetch domains only when current deployment actually changes (not on initial mount/hydration)
  const prevDeploymentIdRef = useRef(project?.currentDeploymentId);
  const mountedRef = useRef(false);
  useEffect(() => {
    mountedRef.current = true;
  }, []);
  useEffect(() => {
    const currentId = project?.currentDeploymentId;
    if (mountedRef.current && currentId && prevDeploymentIdRef.current !== currentId) {
      collection.domains.utils.refetch();
    }
    prevDeploymentIdRef.current = currentId;
  }, [project?.currentDeploymentId]);

  const environmentsQuery = useLiveQuery(
    (q) =>
      q
        .from({ env: collection.environments })
        .where(({ env }) =>
          appId
            ? and(eq(env.projectId, projectId), eq(env.appId, appId))
            : eq(env.projectId, projectId),
        ),
    [projectId, appId],
  );

  const customDomainsQuery = useLiveQuery(
    (q) =>
      q
        .from({ customDomain: collection.customDomains })
        .where(({ customDomain }) =>
          appId
            ? and(eq(customDomain.projectId, projectId), eq(customDomain.appId, appId))
            : eq(customDomain.projectId, projectId),
        )
        .orderBy(({ customDomain }) => customDomain.createdAt, "desc"),
    [projectId, appId],
  );

  const value = useMemo(() => {
    const domains = domainsQuery.data ?? [];
    const deployments = deploymentsQuery.data ?? [];
    const environments = environmentsQuery.data ?? [];
    const customDomains = customDomainsQuery.data ?? [];
    const project = projectQuery.data?.at(0);

    return {
      projectId,
      appId,

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
  }, [
    projectId,
    appId,
    domainsQuery,
    deploymentsQuery,
    projectQuery,
    environmentsQuery,
    customDomainsQuery,
  ]);

  // The projects collection holds every project in the workspace, so once it has
  // finished loading an absent project means it does not exist (or is inaccessible).
  // Checked after all hooks have run to keep hook ordering stable across renders.
  if (!projectQuery.isLoading && !project) {
    notFound();
  }

  return <ProjectDataContext.Provider value={value}>{children}</ProjectDataContext.Provider>;
};

export const useProjectData = () => {
  const context = useContext(ProjectDataContext);
  if (!context) {
    throw new Error("useProjectData must be used within ProjectDataProvider");
  }
  return context;
};

// Asserts an app-scoped context. Use in components that only render under
// /apps/[appId], where appId is always present, to avoid threading optional
// appId fallbacks through them.
export const useAppId = (): string => {
  const { appId } = useProjectData();
  if (!appId) {
    throw new Error("useAppId must be used inside an app-scoped (/apps/[appId]) route");
  }
  return appId;
};
