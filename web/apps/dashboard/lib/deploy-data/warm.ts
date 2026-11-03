"use client";

import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { warmQueries } from "@/lib/collections/warm-queries";
import { getApp } from "@/lib/queries/apps";
import { createLiveQueryCollection } from "@tanstack/react-db";
import type { QueryClient } from "@tanstack/react-query";
import {
  PRODUCTION_LIMIT,
  RECENT_DEPLOYMENTS_LIMIT,
  deploymentDetailOptions,
  recentDeploymentsOptions,
  warmDeploymentsPage,
} from "./deployments";
import {
  customDomainsQueryFor,
  domainsQueryFor,
  environmentsQueryFor,
  policiesQueryFor,
} from "./queries";

// Loads what the app overview reads first, so a click after hovering finds it loaded
export function warmAppPage(queryClient: QueryClient, projectId: string, appId: string): void {
  warmDeploymentsPage(queryClient, projectId, appId);
  void queryClient.prefetchQuery(
    recentDeploymentsOptions(queryClient, projectId, appId, null, RECENT_DEPLOYMENTS_LIMIT),
  );
  const currentDeploymentId = getApp(queryClient, projectId, appId)?.currentDeploymentId;
  if (currentDeploymentId) {
    warmDeployment(queryClient, projectId, currentDeploymentId);
  }
  warmQueries(`app/${projectId}/${appId}`, () => {
    const environments = createLiveQueryCollection(environmentsQueryFor(projectId, appId));
    // Environments load on demand, so the production id is only known once they arrive
    environments
      .preload()
      .then(() => warmProduction(queryClient, projectId, appId))
      .catch(() => {});
    return [
      environments,
      createLiveQueryCollection(domainsQueryFor(projectId, appId)),
      createLiveQueryCollection(customDomainsQueryFor(projectId, appId)),
    ];
  });
}

// Loads one deployment by id, so opening its page after hovering its row finds it loaded
export function warmDeployment(
  queryClient: QueryClient,
  projectId: string,
  deploymentId: string,
): void {
  void queryClient.prefetchQuery(deploymentDetailOptions(queryClient, projectId, deploymentId));
}

function warmProduction(queryClient: QueryClient, projectId: string, appId: string): void {
  const productionId = [...collection.environments.values()].find(
    (e) => e.appId === appId && e.kind === ENVIRONMENT_KIND.production,
  )?.id;
  if (!productionId) {
    return;
  }
  void queryClient.prefetchQuery(
    recentDeploymentsOptions(queryClient, projectId, appId, productionId, PRODUCTION_LIMIT),
  );
  warmQueries(`app/${projectId}/${appId}/production`, () => [
    createLiveQueryCollection(policiesQueryFor(projectId, appId, productionId)),
  ]);
}
