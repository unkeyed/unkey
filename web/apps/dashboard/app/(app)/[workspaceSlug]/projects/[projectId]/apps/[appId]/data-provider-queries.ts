"use client";
import { collection } from "@/lib/collections";
import { DEPLOYMENT_STATUSES } from "@/lib/collections/deploy/deployment-status";
import { DEPLOYMENTS_DEFAULT_LIMIT } from "@/lib/collections/deploy/deployments";
import { warmQueries } from "@/lib/collections/warm-queries";
import {
  type InitialQueryBuilder,
  and,
  createLiveQueryCollection,
  eq,
  inArray,
} from "@tanstack/react-db";

const LISTED_STATUSES = DEPLOYMENT_STATUSES.filter((status) => status !== "skipped");

export const deploymentsQueryFor =
  (projectId: string, appId: string | undefined) => (q: InitialQueryBuilder) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        and(
          appId
            ? and(eq(deployment.projectId, projectId), eq(deployment.appId, appId))
            : eq(deployment.projectId, projectId),
          inArray(deployment.status, LISTED_STATUSES),
        ),
      )
      .orderBy(({ deployment }) => deployment.createdAt, "desc")
      .limit(DEPLOYMENTS_DEFAULT_LIMIT);

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

export function warmAppPage(projectId: string, appId: string) {
  warmQueries(`app/${projectId}/${appId}`, () => [
    createLiveQueryCollection(deploymentsQueryFor(projectId, appId)),
    createLiveQueryCollection(domainsQueryFor(projectId, appId)),
    createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
    createLiveQueryCollection(environmentsQueryFor(projectId, [appId])),
  ]);
}
