"use client";

import { collection } from "@/lib/collections";
import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import {
  type DeploymentStatus,
  isDeploymentSettling,
} from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import { pickPrimaryApp } from "@/lib/collections/deploy/project-cards";
import type { Project } from "@/lib/collections/deploy/projects";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
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
  customDomainsQueryFor,
  deploymentsQueryFor,
  domainsQueryFor,
  environmentsQueryFor,
} from "./data-provider-queries";
import { useAwaitTarget } from "./hooks/use-await-target";

// Deploys arrive from GitHub, the CLI and other people, so every app page
// refreshes on a slow cadence. It speeds up while a row is moving and again
// while this tab waits on an action of its own.
const IDLE_POLL_MS = 20_000;
const SETTLING_POLL_MS = 5_000;
const AWAITING_POLL_MS = 2_000;

function pollIntervalMs(awaiting: boolean, settling: boolean): number {
  if (awaiting) {
    return AWAITING_POLL_MS;
  }
  if (settling) {
    return SETTLING_POLL_MS;
  }
  return IDLE_POLL_MS;
}

type LiveDeploymentTarget = {
  deploymentId: string;
  rolledBack: boolean;
};

type DeploymentStatusTarget = {
  deploymentId: string;
  status: DeploymentStatus;
};

type ProjectDataContextType = {
  projectId: string;
  // Transitional: optional until the detail view and onboarding wizard move
  // under apps/[appId]. Becomes required (no project-wide fallback) once every
  // mount supplies an app.
  appId: string | undefined;

  project:
    | (Project & { repositoryFullName: string | null; currentDeploymentId: string | null })
    | undefined;
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
  awaitDeploymentStatus: (target: DeploymentStatusTarget) => void;
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

  const deploymentsQuery = useLiveQuery(deploymentsQueryFor(projectId, appId), [projectId, appId]);

  const projectQuery = useLiveQuery(
    (q) =>
      q
        .from({ project: collection.projects })
        .where(({ project }) => eq(project.id, projectId))
        .findOne(),
    [projectId],
  );

  const appQuery = useLiveQuery(
    (q) =>
      appId
        ? q
            .from({ app: collection.apps })
            .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId)))
            .findOne()
        : null,
    [projectId, appId],
  );
  const app = appQuery.data;
  const projectAppsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const primaryApp = pickPrimaryApp(projectAppsQuery.data ?? []);
  const currentDeploymentId = appId
    ? app?.currentDeploymentId
    : (primaryApp?.currentDeploymentId ?? undefined);

  const domainsQuery = useLiveQuery(domainsQueryFor(projectId, appId), [projectId, appId]);
  const refetchDeployments = useCallback(() => {
    collection.deployments.utils.refetch();
    trpcUtils.deploy.deployment.list.invalidate();
    trpcUtils.deploy.deployment.listActiveBranches.invalidate();
    trpcUtils.deploy.deployment.listBranches.invalidate();
  }, [trpcUtils]);

  const refetchAll = useCallback(() => {
    collection.projects.utils.refetch();
    collection.apps.utils.refetch();
    refetchDeployments();
    collection.domains.utils.refetch();
    collection.environments.utils.refetch();
    collection.customDomains.utils.refetch();
  }, [refetchDeployments]);

  const liveDeployment = useAwaitTarget<LiveDeploymentTarget>({
    isReached: (target) =>
      app?.currentDeploymentId === target.deploymentId && app.isRolledBack === target.rolledBack,
    onSettled: refetchAll,
  });
  const deploymentStatus = useAwaitTarget<DeploymentStatusTarget>({
    isReached: (target) =>
      deploymentsQuery.data?.find((d) => d.id === target.deploymentId)?.status === target.status,
    onSettled: refetchAll,
  });

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

  const environmentAppIds = useMemo(
    () => (appId ? [appId] : (projectAppsQuery.data ?? []).map((a) => a.id).sort()),
    [appId, projectAppsQuery.data],
  );
  const environmentsQuery = useLiveQuery(
    (q) =>
      environmentAppIds.length === 0 ? null : environmentsQueryFor(projectId, environmentAppIds)(q),
    [projectId, environmentAppIds.join(",")],
  );

  const customDomainsQuery = useLiveQuery(customDomainsQueryFor(projectId, appId), [
    projectId,
    appId,
  ]);

  const hasSettlingDeployment = (deploymentsQuery.data ?? []).some(isDeploymentSettling);
  const hasPendingDomain = (customDomainsQuery.data ?? []).some(
    (d) => d.verificationStatus === "pending" || d.verificationStatus === "verifying",
  );
  useCollectionPolling(refetchDeployments, {
    intervalMs: pollIntervalMs(deploymentStatus.waiting, hasSettlingDeployment),
    enabled: true,
  });
  useCollectionPolling(() => collection.apps.utils.refetch(), {
    intervalMs: pollIntervalMs(liveDeployment.waiting, false),
    enabled: appId !== undefined,
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
    const activeApp = appId ? app : primaryApp;
    const project = projectQuery.data
      ? {
          ...projectQuery.data,
          repositoryFullName: activeApp?.repositoryFullName ?? null,
          currentDeploymentId: activeApp?.currentDeploymentId ?? null,
        }
      : undefined;

    return {
      projectId,
      appId,

      project,
      isProjectLoading: projectQuery.isLoading || projectAppsQuery.isLoading || appQuery.isLoading,

      domains,
      isDomainsLoading: domainsQuery.isLoading,

      deployments,
      isDeploymentsLoading: deploymentsQuery.isLoading,

      environments,
      isEnvironmentsLoading: projectAppsQuery.isLoading || environmentsQuery.isLoading,

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
      awaitLiveDeployment: liveDeployment.start,
      awaitDeploymentStatus: deploymentStatus.start,
    };
  }, [
    projectId,
    appId,
    domainsQuery,
    deploymentsQuery,
    projectQuery,
    projectAppsQuery.isLoading,
    appQuery.isLoading,
    primaryApp,
    app,
    environmentsQuery,
    customDomainsQuery,
    refetchDeployments,
    refetchAll,
    liveDeployment.start,
    deploymentStatus.start,
  ]);

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
