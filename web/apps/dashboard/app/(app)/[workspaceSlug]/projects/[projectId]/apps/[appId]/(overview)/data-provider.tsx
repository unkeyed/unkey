"use client";

import { collection } from "@/lib/collections";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import { isDeploymentSettling } from "@/lib/collections/deploy/deployment-status";
import { DEPLOYMENTS_DEFAULT_LIMIT, type Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import type { Project } from "@/lib/collections/deploy/projects";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { notFound, useParams } from "next/navigation";
import {
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from "react";
import {
  type LiveDeploymentTarget,
  useAwaitLiveDeployment,
} from "./hooks/use-await-live-deployment";

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
  awaitLiveDeployment: (target: LiveDeploymentTarget) => void;
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

  const trpcUtils = trpc.useUtils();

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
  const appQuery = useLiveQuery(
    (q) =>
      appId
        ? q
            .from({ app: collection.apps })
            .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId)))
        : null,
    [projectId, appId],
  );
  const app = appQuery.data?.at(0);
  const currentDeploymentId = appId ? app?.currentDeploymentId : project?.currentDeploymentId;

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
  const refetchDeployments = useCallback(() => {
    collection.deployments.utils.refetch();
    trpcUtils.deploy.deployment.invalidate();
  }, [trpcUtils]);

  const refetchAll = useCallback(() => {
    collection.projects.utils.refetch();
    collection.apps.utils.refetch();
    refetchDeployments();
    collection.domains.utils.refetch();
    collection.environments.utils.refetch();
    collection.customDomains.utils.refetch();
  }, [refetchDeployments]);

  const awaitLiveDeployment = useAwaitLiveDeployment(app, refetchAll);

  // refetch domains only when current deployment actually changes (not on initial mount/hydration)
  const prevDeploymentIdRef = useRef(currentDeploymentId);
  const mountedRef = useRef(false);
  useEffect(() => {
    mountedRef.current = true;
  }, []);
  useEffect(() => {
    if (
      mountedRef.current &&
      currentDeploymentId &&
      prevDeploymentIdRef.current !== currentDeploymentId
    ) {
      collection.domains.utils.refetch();
    }
    prevDeploymentIdRef.current = currentDeploymentId;
  }, [currentDeploymentId]);

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

  const hasSettlingDeployment = (deploymentsQuery.data ?? []).some(isDeploymentSettling);
  const hasPendingDomain = (customDomainsQuery.data ?? []).some(
    (d) => d.verificationStatus === "pending" || d.verificationStatus === "verifying",
  );
  useCollectionPolling(() => collection.deployments.utils.refetch(), {
    intervalMs: 5000,
    enabled: hasSettlingDeployment,
  });
  useCollectionPolling(() => collection.customDomains.utils.refetch(), {
    intervalMs: 5000,
    enabled: hasPendingDomain,
  });

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
      refetchDeployments,
      refetchCustomDomains: () => collection.customDomains.utils.refetch(),
      refetchAll,
      awaitLiveDeployment,
    };
  }, [
    projectId,
    appId,
    domainsQuery,
    deploymentsQuery,
    projectQuery,
    environmentsQuery,
    customDomainsQuery,
    refetchDeployments,
    refetchAll,
    awaitLiveDeployment,
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
