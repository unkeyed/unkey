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
    (q) =>
      enabled
        ? q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId))
        : null,
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

  const recentIds = new Set((recentDeployments.data ?? []).map((d) => d.id));
  const currentOutsideWindowIds = recentDeployments.isLoading
    ? []
    : (projectApps.data ?? [])
        .flatMap((app) =>
          app.currentDeploymentId && !recentIds.has(app.currentDeploymentId)
            ? [app.currentDeploymentId]
            : [],
        )
        .sort();
  const currentOutsideWindow = useLiveQuery(
    (q) =>
      currentOutsideWindowIds.length === 0
        ? null
        : q
            .from({ deployment: collection.deployments })
            .where(({ deployment }) =>
              and(
                eq(deployment.projectId, projectId),
                inArray(deployment.id, currentOutsideWindowIds),
              ),
            ),
    [projectId, currentOutsideWindowIds.join(",")],
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
        deployments: [
          ...new Map(
            [...(recentDeployments.data ?? []), ...(currentOutsideWindow.data ?? [])].map((d) => [
              d.id,
              d,
            ]),
          ).values(),
        ],
        productionDomains: productionDomains.data ?? [],
      }),
    [
      projectId,
      projectApps.data,
      recentDeployments.data,
      currentOutsideWindow.data,
      productionDomains.data,
    ],
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
