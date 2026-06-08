"use client";

import { LoadingState } from "@/components/loading-state";
import { useAppSlug, useProjectSlug } from "@/hooks/use-route-slugs";
import { collection } from "@/lib/collections";
import type { App } from "@/lib/collections/deploy/apps";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import { DEPLOYMENTS_DEFAULT_LIMIT, type Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Project } from "@/lib/collections/deploy/projects";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Empty } from "@unkey/ui";
import {
  type PropsWithChildren,
  type ReactNode,
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from "react";

type ProjectDataContextType = {
  projectId: string;
  projectSlug: string;
  // Transitional: optional until the detail view and onboarding wizard move
  // under apps/[appSlug]. Becomes required (no project-wide fallback) once every
  // mount supplies an app.
  appId: string | undefined;
  appSlug: string | undefined;

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

// Route mounts resolve via [projectSlug]/[appSlug] params; the onboarding
// wizard mounts with ids it got back from create mutations.
type ProjectDataProviderProps = PropsWithChildren<{
  projectId?: string;
  appId?: string;
}>;

// Resolves route slugs (or wizard-provided ids) to full rows before mounting
// the data queries, so everything below the provider keeps a guaranteed
// projectId (and appId when app-scoped).
export const ProjectDataProvider = ({
  children,
  projectId: projectIdProp,
  appId: appIdProp,
}: ProjectDataProviderProps) => {
  const projectSlug = useProjectSlug();
  const appSlug = useAppSlug();

  if (!projectIdProp && !projectSlug) {
    throw new Error("ProjectDataProvider requires a projectId prop or a [projectSlug] route param");
  }

  const projectQuery = useLiveQuery(
    (q) =>
      q
        .from({ project: collection.projects })
        .where(({ project }) =>
          projectIdProp ? eq(project.id, projectIdProp) : eq(project.slug, projectSlug ?? ""),
        ),
    [projectIdProp, projectSlug],
  );
  const project = projectQuery.data?.at(0);

  if (!project) {
    return projectQuery.isLoading ? (
      <LoadingState />
    ) : (
      <NotFound
        title="Project not found"
        description={`No project "${projectIdProp ?? projectSlug}" in this workspace`}
      />
    );
  }

  if (!appIdProp && !appSlug) {
    return <ResolvedProjectDataProvider project={project}>{children}</ResolvedProjectDataProvider>;
  }

  return (
    <AppResolver project={project} appId={appIdProp} appSlug={appSlug}>
      {children}
    </AppResolver>
  );
};

export const useProjectData = () => {
  const context = useContext(ProjectDataContext);
  if (!context) {
    throw new Error("useProjectData must be used within ProjectDataProvider");
  }
  return context;
};

// Asserts an app-scoped context. Use in components that only render under
// /apps/[appSlug], where appId is always present, to avoid threading optional
// appId fallbacks through them.
export const useAppId = (): string => {
  const { appId } = useProjectData();
  if (!appId) {
    throw new Error("useAppId must be used inside an app-scoped (/apps/[appSlug]) route");
  }
  return appId;
};

// Mounted only when the project is known: the apps collection requires a
// projectSlug filter (app slugs are only unique per project).
const AppResolver = ({
  children,
  project,
  appId,
  appSlug,
}: PropsWithChildren<{ project: Project; appId?: string; appSlug?: string }>) => {
  const appQuery = useLiveQuery(
    (q) =>
      q.from({ app: collection.apps }).where(({ app }) => {
        const inProject = eq(app.projectSlug, project.slug);
        if (appId) {
          return and(inProject, eq(app.id, appId));
        }
        if (appSlug) {
          return and(inProject, eq(app.slug, appSlug));
        }
        // ProjectDataProvider only mounts AppResolver with an appId or appSlug.
        console.warn("[AppResolver] mounted without appId or appSlug");
        return and(inProject, eq(app.slug, ""));
      }),
    [project.slug, appId, appSlug],
  );
  const app = appQuery.data?.at(0);

  if (!app) {
    return appQuery.isLoading ? (
      <LoadingState />
    ) : (
      <NotFound
        title="App not found"
        description={`No app "${appId ?? appSlug}" in this project`}
      />
    );
  }

  return (
    <ResolvedProjectDataProvider project={project} app={app}>
      {children}
    </ResolvedProjectDataProvider>
  );
};

const ResolvedProjectDataProvider = ({
  children,
  project,
  app,
}: PropsWithChildren<{ project: Project; app?: App }>) => {
  const projectId = project.id;
  const appId = app?.id;

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
  const prevDeploymentIdRef = useRef(project.currentDeploymentId);
  const mountedRef = useRef(false);
  useEffect(() => {
    mountedRef.current = true;
  }, []);
  useEffect(() => {
    const currentId = project.currentDeploymentId;
    if (mountedRef.current && currentId && prevDeploymentIdRef.current !== currentId) {
      collection.domains.utils.refetch();
    }
    prevDeploymentIdRef.current = currentId;
  }, [project.currentDeploymentId]);

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

    return {
      projectId,
      projectSlug: project.slug,
      appId,
      appSlug: app?.slug,

      project,
      isProjectLoading: false,

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
    project,
    app,
    domainsQuery,
    deploymentsQuery,
    environmentsQuery,
    customDomainsQuery,
  ]);

  return <ProjectDataContext.Provider value={value}>{children}</ProjectDataContext.Provider>;
};

const NotFound = ({ title, description }: { title: string; description: ReactNode }) => (
  <div className="w-full min-h-[60vh] flex justify-center items-center">
    <Empty>
      <Empty.Title>{title}</Empty.Title>
      <Empty.Description>{description}</Empty.Description>
    </Empty>
  </div>
);
