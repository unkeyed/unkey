"use client";
import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import {
  DEPLOYMENT_STATUSES,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import { buildProjectApps } from "@/lib/collections/deploy/project-cards";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { and, eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";

const RECENT_DEPLOYMENTS_PER_PROJECT = 10;
const IN_FLIGHT_POLL_MS = 5_000;

export function useProjectCard(projectId: string, { nearViewport }: { nearViewport: boolean }) {
  const enabled = nearViewport && projectId !== SERVER_PLACEHOLDER;

  const projectApps = useLiveQuery(
    (q) => {
      if (!enabled) {
        return null;
      }
      // The filter lives in a subquery so it reaches the deployments load. On the
      // nullable side of the join, an outer where would not.
      const projectDeployments = q
        .from({ current: collection.deployments })
        .where(({ current }) => eq(current.projectId, projectId));
      return q
        .from({ app: collection.apps })
        .where(({ app }) => eq(app.projectId, projectId))
        .leftJoin({ current: projectDeployments }, ({ app, current }) =>
          eq(app.currentDeploymentId, current.id),
        );
    },
    [projectId, enabled],
  );
  const recentDeployments = useLiveQuery(
    (q) =>
      enabled
        ? q
            .from({ deployment: collection.deployments })
            .where(({ deployment }) =>
              and(
                eq(deployment.projectId, projectId),
                inArray(deployment.status, [...DEPLOYMENT_STATUSES]),
              ),
            )
            .orderBy(({ deployment }) => deployment.createdAt, "desc")
            .limit(RECENT_DEPLOYMENTS_PER_PROJECT)
        : null,
    [projectId, enabled],
  );

  const productionDomains = useLiveQuery(
    (q) =>
      enabled
        ? q
            .from({ domain: collection.productionDomains })
            .where(({ domain }) => eq(domain.projectId, projectId))
        : null,
    [projectId, enabled],
  );

  const apps = useMemo(
    () =>
      buildProjectApps(projectId, {
        apps: projectApps.data ?? [],
        recentDeployments: recentDeployments.data ?? [],
        productionDomains: productionDomains.data ?? [],
      }),
    [projectId, projectApps.data, recentDeployments.data, productionDomains.data],
  );

  const inFlight = apps.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(
    () =>
      Promise.all([
        queryClient.refetchQueries({ queryKey: ["apps", projectId], type: "active" }),
        queryClient.refetchQueries({ queryKey: ["deployments", projectId], type: "active" }),
      ]),
    { intervalMs: IN_FLIGHT_POLL_MS, enabled: enabled && inFlight },
  );

  return {
    apps,
    isLoading: !nearViewport || (enabled && (projectApps.isLoading || recentDeployments.isLoading)),
  };
}
