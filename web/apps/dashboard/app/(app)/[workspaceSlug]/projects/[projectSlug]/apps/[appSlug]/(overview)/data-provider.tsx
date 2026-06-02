"use client";

import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
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
  // under apps/[appSlug]. Becomes required (no project-wide fallback) once every
  // mount supplies an app.
  appId: string | undefined;

  // URL-facing identifiers, for building links and breadcrumbs.
  projectSlug: string | undefined;
  appSlug: string | undefined;

  project: Project | undefined;
  app: App | undefined;
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
  const projectSlug = typeof params?.projectSlug === "string" ? params.projectSlug : undefined;
  const appSlug = typeof params?.appSlug === "string" ? params.appSlug : undefined;

  // Resolve the project from either an explicit id prop (onboarding wizard) or
  // the [projectSlug] route param. The slug is unique per workspace, so the
  // globally-fetched projects collection resolves it without a scoping filter.
  const projectQuery = useLiveQuery(
    (q) => {
      if (projectIdProp) {
        return q
          .from({ project: collection.projects })
          .where(({ project }) => eq(project.id, projectIdProp));
      }
      if (projectSlug) {
        return q
          .from({ project: collection.projects })
          .where(({ project }) => eq(project.slug, projectSlug));
      }
      return undefined;
    },
    [projectIdProp, projectSlug],
  );

  const project = projectQuery.data?.at(0);
  const projectId = projectIdProp ?? project?.id;

  // Resolve the app once the project id is known. The apps collection requires
  // a projectId filter, so this query stays disabled until projectId resolves.
  const appQuery = useLiveQuery(
    (q) => {
      if (!projectId) {
        return undefined;
      }
      if (appIdProp) {
        return q
          .from({ app: collection.apps })
          .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appIdProp)));
      }
      if (appSlug) {
        return q
          .from({ app: collection.apps })
          .where(({ app }) => and(eq(app.projectId, projectId), eq(app.slug, appSlug)));
      }
      return undefined;
    },
    [projectId, appIdProp, appSlug],
  );

  const app = appQuery.data?.at(0);
  const appId = appIdProp ?? app?.id;

  const deploymentsQuery = useLiveQuery(
    (q) =>
      projectId
        ? q
            .from({ deployment: collection.deployments })
            .where(({ deployment }) =>
              appId
                ? and(eq(deployment.projectId, projectId), eq(deployment.appId, appId))
                : eq(deployment.projectId, projectId),
            )
            .orderBy(({ deployment }) => deployment.createdAt, "desc")
            .limit(DEPLOYMENTS_DEFAULT_LIMIT)
        : undefined,
    [projectId, appId],
  );

  const domainsQuery = useLiveQuery(
    (q) =>
      projectId
        ? q
            .from({ domain: collection.domains })
            .where(({ domain }) =>
              appId
                ? and(eq(domain.projectId, projectId), eq(domain.appId, appId))
                : eq(domain.projectId, projectId),
            )
            .orderBy(({ domain }) => domain.createdAt, "desc")
        : undefined,
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
      projectId
        ? q
            .from({ env: collection.environments })
            .where(({ env }) =>
              appId
                ? and(eq(env.projectId, projectId), eq(env.appId, appId))
                : eq(env.projectId, projectId),
            )
        : undefined,
    [projectId, appId],
  );

  const customDomainsQuery = useLiveQuery(
    (q) =>
      projectId
        ? q
            .from({ customDomain: collection.customDomains })
            .where(({ customDomain }) =>
              appId
                ? and(eq(customDomain.projectId, projectId), eq(customDomain.appId, appId))
                : eq(customDomain.projectId, projectId),
            )
            .orderBy(({ customDomain }) => customDomain.createdAt, "desc")
        : undefined,
    [projectId, appId],
  );

  const value = useMemo<ProjectDataContextType | null>(() => {
    if (!projectId) {
      return null;
    }
    const domains = domainsQuery.data ?? [];
    const deployments = deploymentsQuery.data ?? [];
    const environments = environmentsQuery.data ?? [];
    const customDomains = customDomainsQuery.data ?? [];

    return {
      projectId,
      appId,

      projectSlug: projectSlug ?? project?.slug,
      appSlug: appSlug ?? app?.slug,

      project,
      app,
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
    projectSlug,
    appSlug,
    project,
    app,
    domainsQuery,
    deploymentsQuery,
    projectQuery,
    environmentsQuery,
    customDomainsQuery,
  ]);

  // Slug routes resolve asynchronously through the collections; an explicit id
  // prop (wizard) resolves synchronously and never 404s. Gate notFound on
  // isReady (collection has settled) rather than !isLoading, so a cold-cache
  // first render doesn't flash a spurious 404 before the fetch lands.
  const isResolvingProject = !projectIdProp && Boolean(projectSlug) && !projectQuery.isReady;
  const projectMissing = !projectIdProp && Boolean(projectSlug) && projectQuery.isReady && !project;

  const isResolvingApp = !appIdProp && Boolean(appSlug) && (!projectId || !appQuery.isReady);
  const appMissing =
    !appIdProp && Boolean(appSlug) && Boolean(projectId) && appQuery.isReady && !app;

  if (projectMissing || appMissing) {
    notFound();
  }

  if (!value || isResolvingProject || isResolvingApp) {
    return null;
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
