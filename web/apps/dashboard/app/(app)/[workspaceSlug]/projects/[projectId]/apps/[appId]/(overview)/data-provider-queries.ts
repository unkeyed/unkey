"use client";
import { collection } from "@/lib/collections";
import {
  DEFAULT_DEPLOYMENT_STATUS_GROUPS,
  DEPLOYMENT_STATUSES,
} from "@/lib/collections/deploy/deployment-status";
import { warmQueries } from "@/lib/collections/warm-queries";
import {
  type InitialQueryBuilder,
  and,
  createLiveQueryCollection,
  eq,
  gte,
  inArray,
  lt,
} from "@tanstack/react-db";
import type { DeploymentListFilter } from "./deployments/hooks/deployment-list-input";
import { buildDeploymentListInput } from "./deployments/hooks/deployment-list-input";

export const domainsQueryFor =
  (projectId: string, appId: string | undefined) => (q: InitialQueryBuilder) =>
    q
      .from({ domain: collection.domains })
      .where(({ domain }) =>
        appId
          ? and(eq(domain.projectId, projectId), eq(domain.appId, appId))
          : eq(domain.projectId, projectId),
      )
      .orderBy(({ domain }) => domain.createdAt, "desc");

export const appDeploymentQueryFor =
  (projectId: string, appId: string, deploymentId: string) => (q: InitialQueryBuilder) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        and(
          eq(deployment.projectId, projectId),
          eq(deployment.appId, appId),
          eq(deployment.id, deploymentId),
        ),
      );

export const customDomainsQueryFor =
  (projectId: string, appId: string | undefined) => (q: InitialQueryBuilder) =>
    q
      .from({ customDomain: collection.customDomains })
      .where(({ customDomain }) =>
        appId
          ? and(eq(customDomain.projectId, projectId), eq(customDomain.appId, appId))
          : eq(customDomain.projectId, projectId),
      )
      .orderBy(({ customDomain }) => customDomain.createdAt, "desc");

export const environmentsQueryFor =
  (projectId: string, appIds: string[]) => (q: InitialQueryBuilder) =>
    q
      .from({ env: collection.environments })
      .where(({ env }) => and(eq(env.projectId, projectId), inArray(env.appId, appIds)));

// Skipped pushes never deployed, so the shared views leave them out
const RECENT_STATUSES = DEPLOYMENT_STATUSES.filter((status) => status !== "skipped");
const RECENT_LIMIT = 100;
export const DEPLOYMENTS_PAGE_SIZE = 25;

// The newest deployments of an app, or of the whole project without one
export const recentDeploymentsQueryFor =
  (projectId: string, appId: string | undefined) => (q: InitialQueryBuilder) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        and(
          appId
            ? and(eq(deployment.projectId, projectId), eq(deployment.appId, appId))
            : eq(deployment.projectId, projectId),
          inArray(deployment.status, RECENT_STATUSES),
        ),
      )
      .orderBy(({ deployment }) => deployment.createdAt, "desc")
      .limit(RECENT_LIMIT);

export const deploymentsTableQueryFor =
  (projectId: string, appId: string, filter: DeploymentListFilter) => (q: InitialQueryBuilder) => {
    let query = q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        and(
          eq(deployment.projectId, projectId),
          eq(deployment.appId, appId),
          inArray(deployment.status, filter.statuses),
        ),
      );
    const { environmentId, branches, startTime, endTime } = filter;
    if (environmentId !== undefined) {
      query = query.where(({ deployment }) => eq(deployment.environmentId, environmentId));
    }
    if (branches.length > 0) {
      query = query.where(({ deployment }) => inArray(deployment.gitBranch, branches));
    }
    if (startTime !== undefined) {
      query = query.where(({ deployment }) => gte(deployment.createdAt, startTime));
    }
    if (endTime !== undefined) {
      query = query.where(({ deployment }) => lt(deployment.createdAt, endTime));
    }
    return query.orderBy(({ deployment }) => deployment.createdAt, "desc");
  };

// Warms the table's unfiltered first page, which is what a visit without filter
// params in the url shows. The limit matches the first subset useLiveInfiniteQuery loads
export function warmDeploymentsTable(projectId: string, appId: string) {
  const { filter } = buildDeploymentListInput(
    DEFAULT_DEPLOYMENT_STATUS_GROUPS.map((value) => ({
      id: value,
      field: "status" as const,
      operator: "is" as const,
      value,
    })),
    [],
  );
  warmQueries(`deployments/${projectId}/${appId}`, () => [
    createLiveQueryCollection((q) =>
      deploymentsTableQueryFor(projectId, appId, filter)(q).limit(DEPLOYMENTS_PAGE_SIZE),
    ),
  ]);
}

export function warmAppPage(projectId: string, appId: string) {
  warmDeploymentsTable(projectId, appId);
  const currentDeploymentId = collection.apps.get(appId)?.currentDeploymentId;
  warmQueries(`app/${projectId}/${appId}`, () => [
    createLiveQueryCollection(recentDeploymentsQueryFor(projectId, appId)),
    createLiveQueryCollection(domainsQueryFor(projectId, appId)),
    createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
    createLiveQueryCollection(environmentsQueryFor(projectId, [appId])),
    ...(currentDeploymentId
      ? [createLiveQueryCollection(appDeploymentQueryFor(projectId, appId, currentDeploymentId))]
      : []),
  ]);
}
